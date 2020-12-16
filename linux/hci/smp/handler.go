package smp

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/photostorm/ble/linux/hci"
)

//func smpOnPairingRequest(c *Conn, in pdu) ([]byte, error) {
//	if len(in) < 6 {
//		return nil, fmt.Errorf("%v, invalid length %v", hex.EncodeToString(in), len(in))
//	}
//
//	rx := smpConfig{}
//	rx.ioCap = in[0]
//	rx.oobFlag = in[1]
//	rx.authReq = in[2]
//	rx.maxKeySz = in[3]
//	rx.initKeyDist = in[4]
//	rx.respKeyDist = in[5]
//
//	return nil, nil
//}

func smpOnPairingResponse(t *transport, in pdu) ([]byte, error) {
	if len(in) < 6 {
		return nil, fmt.Errorf("%v, invalid length %v", hex.EncodeToString(in), len(in))
	}

	rx := hci.SmpConfig{}
	rx.IoCap = in[0]
	rx.OobFlag = in[1]
	rx.AuthReq = in[2]
	rx.MaxKeySize = in[3]
	rx.InitKeyDist = in[4]
	rx.RespKeyDist = in[5]
	t.pairing.response = rx

	t.pairing.pairingType = JustWorks
	t.pairing.passKeyIteration = 0

	t.pairing.legacy = isLegacy(rx.AuthReq)
	t.pairing.pairingType = determinePairingType(t)

	println("detected pairing type: ", pairingTypeStrings[t.pairing.pairingType])

	if t.pairing.pairingType == Oob &&
		len(t.pairing.authData.OOBData) == 0 {
		t.pairing.state = Error
		return nil, errors.New("pairing requires OOB data but OOB data not specified")
	}

	if t.pairing.legacy {
		return nil, t.sendMConfirm()
	}

	return nil, t.sendPublicKey()
}

func smpOnPairingConfirm(t *transport, in pdu) ([]byte, error) {
	if t.pairing == nil {
		return nil, errors.New("no pairing context")
	}

	if len(in) != 16 {
		return nil, errors.New("invalid length")
	}

	t.pairing.remoteConfirm = in

	err := t.sendPairingRandom()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func smpOnPairingRandom(t *transport, in pdu) ([]byte, error) {
	if t.pairing == nil {
		return nil, errors.New("no pairing context")
	}

	if len(in) != 16 {
		return nil, errors.New("invalid length")
	}

	t.pairing.remoteRandom = in

	//conf check
	if t.pairing.legacy {
		return onLegacyRandom(t)
	}

	return onSecureRandom(t)
}

func onSecureRandom(t *transport) ([]byte, error) {
	if t.pairing.pairingType == Passkey {
		more, err := handlePassKeyRandom(t)
		if err != nil {
			return nil, err
		}

		if more {
			return nil, nil
		}
	} else {
		err := t.pairing.checkConfirm()
		if err != nil {
			return nil, err
		}
	}

	// TODO
	// here we would do the compare from g2(...) but this is just works only for now
	// move on to auth stage 2 (2.3.5.6.5) calc mackey, ltk
	err := t.pairing.calcMacLtk()
	if err != nil {
		return nil, err
	}

	//send dhkey check
	err = t.sendDHKeyCheck()
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func onLegacyRandom(t *transport) ([]byte, error) {
	err := t.pairing.checkLegacyConfirm()
	if err != nil {
		return nil, err
	}

	lRand := t.pairing.localRandom
	rRand := t.pairing.remoteRandom

	//calculate STK
	k := getLegacyParingTK(t.pairing.authData.Passkey)
	stk, err := smpS1(k, rRand, lRand)
	t.pairing.shortTermKey = stk

	if t.pairing.request.AuthReq & 0x01 == 0x00 {
		t.pairing.state = Finished
	}

	err = t.encrypter.Encrypt()
	return nil, err
}

func smpOnPairingPublicKey(t *transport, in pdu) ([]byte, error) {
	if t.pairing == nil {
		return nil, errors.New("no pairing context")
	}

	if len(in) != 64 {
		return nil, errors.New("invalid length")
	}

	pubk, ok := UnmarshalPublicKey(in)

	if !ok {
		return nil, errors.New("key error")
	}

	t.pairing.scRemotePubKey = pubk

	if t.pairing.pairingType == Passkey {
		startPassKeyPairing(t)
	}
	return nil, nil
}

func smpOnDHKeyCheck(t *transport, in pdu) ([]byte, error) {
	if t.pairing == nil {
		return nil, errors.New("no pairing context")
	}

	t.pairing.scRemoteDHKeyCheck = in
	err := t.pairing.checkDHKeyCheck()
	if err != nil {
		//dhkeycheck failed!
		return nil, err
	}

	println("dhkey check pass!")
	err = t.saveBondInfo()
	if err != nil {
		return nil, err
	}

	//at this point, the pairing is complete
	t.pairing.state = Finished


	//todo: separate this out
	return nil, t.encrypter.Encrypt()
}

func smpOnPairingFailed(t *transport, in pdu) ([]byte, error) {
	reason := "unknown"
	if len(in) > 0 {
		reason = pairingFailedReason[in[0]]
	}
	return nil, fmt.Errorf("pairing failed: %s", reason)
}

func smpOnSecurityRequest(t *transport, in pdu) ([]byte, error) {
	if len(in) < 1 {
		return nil, fmt.Errorf("%v, invalid length %v", hex.EncodeToString(in), len(in))
	}

	ra := hex.EncodeToString(t.pairing.remoteAddr)
	bi, err := t.bondManager.Find(ra)

	if err == nil {
		t.pairing.bond = bi
		return nil, t.encrypter.Encrypt()
	}

	//todo: clean this up
	rx := hci.SmpConfig{}
	rx.AuthReq = in[0]

	//match the incoming request parameters
	t.pairing.request.AuthReq = rx.AuthReq
	//no bonding information stored, so trigger a bond
	return nil, t.sendPairingRequest()
}

func smpOnEncryptionInformation(t *transport, in pdu) ([]byte, error) {
	//need to save the ltk, ediv, and rand to a file
	t.pairing.bond = hci.NewBondInfo(in, 0, 0, true)

	return nil, nil
}

func smpOnMasterIdentification(t *transport, in pdu) ([]byte, error) {
	data := []byte(in)
	ediv := binary.LittleEndian.Uint16(data[:2])
	randVal := binary.LittleEndian.Uint64(data[2:])

	ltk := t.pairing.bond.LongTermKey()
	t.pairing.bond = hci.NewBondInfo(ltk, ediv, randVal, true)

	err := t.saveBondInfo()
	if err != nil {
		return nil, err
	}

	t.pairing.state = Finished

	return nil, nil
}

func handlePassKeyRandom(t *transport) (bool, error) {
	err := t.pairing.checkPasskeyConfirm()
	if err != nil {
		return false, err
	}

	t.pairing.passKeyIteration++

	if t.pairing.passKeyIteration < passkeyIterationCount {
		continuePassKeyPairing(t)
		return true, nil
	}

	return false, nil
}

func startPassKeyPairing(t *transport) {

	t.pairing.passKeyIteration = 0

	continuePassKeyPairing(t)
}

func continuePassKeyPairing(t *transport) {
	confirm, random := t.pairing.generatePassKeyConfirm()
	t.pairing.localRandom = random
	out := append([]byte{pairingConfirm}, confirm...)
	t.send(out)
}

//Core spec v5.0 Vol 3, Part H, 2.3.5.1
//Tables 2.6, 2.7, and 2.8
var ioCapsTableSC = [][]int{
	{ JustWorks, JustWorks, Passkey, JustWorks, Passkey },
	{ JustWorks, NumericComp, Passkey, JustWorks, NumericComp },
	{ Passkey, Passkey, Passkey, JustWorks, Passkey },
	{ JustWorks, JustWorks, JustWorks, JustWorks, JustWorks },
	{ Passkey, NumericComp, Passkey, JustWorks, NumericComp },
}

var ioCapsTableLegacy = [][]int{
	{ JustWorks, JustWorks, Passkey, JustWorks, Passkey },
	{ JustWorks, JustWorks, Passkey, JustWorks, Passkey },
	{ Passkey, Passkey, Passkey, JustWorks, Passkey },
	{ JustWorks, JustWorks, JustWorks, JustWorks, JustWorks },
	{ Passkey, Passkey, Passkey, JustWorks, Passkey },
}

func determinePairingType(t *transport) int {
	mitmMask := byte(0x04)

	req := t.pairing.request
	rsp := t.pairing.response


	if req.OobFlag == 0x01 && rsp.OobFlag == 0x01 && t.pairing.legacy {
		return Oob
	}

	if req.OobFlag == 0x01 || rsp.OobFlag == 0x01 {
		return Oob
	}

	if req.AuthReq & mitmMask == 0x00 &&
		rsp.AuthReq & mitmMask == 0x00 {
		return JustWorks
	}

	pairingTypeTable := ioCapsTableSC
	if t.pairing.legacy {
		pairingTypeTable = ioCapsTableLegacy
	}

	if rsp.IoCap >= hci.IoCapsReservedStart ||
		req.IoCap >= hci.IoCapsReservedStart {
		//todo: is this a valid assumption or should this return an error instead?
		return JustWorks
	}
	return pairingTypeTable[rsp.IoCap][req.IoCap]
}
