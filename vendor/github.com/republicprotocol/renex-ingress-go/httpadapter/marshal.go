package httpadapter

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/republicprotocol/renex-ingress-go/ingress"
	"github.com/republicprotocol/republic-go/order"
)

// OrderFragment is an order.EncryptedFragment, encrypted by the trader. It
// stores the an index that identifies which index of shamir.Shares are stored
// in the OrderFragment. It is represented as a JSON object. This
// representation is useful for HTTP drivers.
type OrderFragment struct {
	OrderSignature  string           `json:"orderSignature"`
	OrderID         string           `json:"orderId"`
	OrderType       order.Type       `json:"orderType"`
	OrderParity     order.Parity     `json:"orderParity"`
	OrderSettlement order.Settlement `json:"orderSettlement"`
	OrderExpiry     int64            `json:"orderExpiry"`
	Index           int64            `json:"index"`
	ID              string           `json:"id"`
	EpochDepth      int32            `json:"epochDepth"`
	Tokens          string           `json:"tokens"`
	Price           []string         `json:"price"`
	Volume          []string         `json:"volume"`
	MinimumVolume   []string         `json:"minimumVolume"`
	Nonce           string           `json:"nonce"`
}

// OrderFragments is a slice.
type OrderFragments []OrderFragment

// An OrderFragmentMapping maps pods to encrypted order fragments represented
// as a JSON object. This representation is useful for HTTP drivers.
type OrderFragmentMapping map[string]OrderFragments

// OrderFragmentMappings is a slice where the index of an OrderFragmentMapping
// represents the epoch depth of each OrderFragment inside the mapping.
type OrderFragmentMappings []OrderFragmentMapping

// OpenOrderRequest is an JSON object sent to the HTTP handlers to request the
// opening of an order.
type OpenOrderRequest struct {
	Signature             string                `json:"signature"`
	OrderFragmentMappings OrderFragmentMappings `json:"orderFragmentMappings"`
}

func MarshalSignature(signatureIn [65]byte) string {
	return base64.StdEncoding.EncodeToString(signatureIn[:])
}

func MarshalOrderID(orderIDIn order.ID) string {
	return base64.StdEncoding.EncodeToString(orderIDIn[:])
}

func MarshalOrderFragmentID(orderFragmentIDIn order.FragmentID) string {
	return base64.StdEncoding.EncodeToString(orderFragmentIDIn[:])
}

func MarshalEncryptedCoExpShare(valueIn order.EncryptedCoExpShare) []string {
	return []string{
		base64.StdEncoding.EncodeToString(valueIn.Co),
		base64.StdEncoding.EncodeToString(valueIn.Exp),
	}
}

func MarshalOrderFragment(orderFragmentIn ingress.OrderFragment) OrderFragment {
	orderFragment := OrderFragment{}
	orderFragment.Index = orderFragmentIn.Index
	orderFragment.OrderID = MarshalOrderID(orderFragmentIn.OrderID)
	orderFragment.OrderType = orderFragmentIn.OrderType
	orderFragment.OrderParity = orderFragmentIn.OrderParity
	orderFragment.OrderSettlement = orderFragmentIn.OrderSettlement
	orderFragment.OrderExpiry = orderFragmentIn.OrderExpiry.Unix()
	orderFragment.ID = MarshalOrderFragmentID(orderFragmentIn.ID)
	orderFragment.Tokens = base64.StdEncoding.EncodeToString(orderFragmentIn.Tokens)
	orderFragment.Price = MarshalEncryptedCoExpShare(orderFragmentIn.Price)
	orderFragment.Volume = MarshalEncryptedCoExpShare(orderFragmentIn.Volume)
	orderFragment.MinimumVolume = MarshalEncryptedCoExpShare(orderFragmentIn.MinimumVolume)
	orderFragment.Nonce = base64.StdEncoding.EncodeToString(orderFragmentIn.Nonce)
	return orderFragment
}

func UnmarshalSignature(signatureIn string) ([65]byte, error) {
	signature := [65]byte{}
	signatureBytes, err := base64.StdEncoding.DecodeString(signatureIn)
	if err != nil {
		return signature, fmt.Errorf("cannot decode signature %v: %v", signatureIn, err)
	}
	if len(signatureBytes) != 65 {
		return signature, ErrInvalidSignatureLength
	}
	copy(signature[:], signatureBytes)
	return signature, nil
}

func UnmarshalOrderID(orderIDIn string) (order.ID, error) {
	orderID := order.ID{}
	orderIDBytes, err := base64.StdEncoding.DecodeString(orderIDIn)
	if err != nil {
		return orderID, fmt.Errorf("cannot decode order id %v: %v", orderIDIn, err)
	}
	if len(orderIDBytes) != 32 {
		return orderID, ErrInvalidOrderIDLength
	}
	copy(orderID[:], orderIDBytes)
	return orderID, nil
}

func UnmarshalOrderFragmentID(orderFragmentIDIn string) (order.FragmentID, error) {
	orderFragmentID := order.FragmentID{}
	orderFragmentIDBytes, err := base64.StdEncoding.DecodeString(orderFragmentIDIn)
	if err != nil {
		return orderFragmentID, fmt.Errorf("cannot decode order fragment id %v: %v", orderFragmentIDIn, err)
	}
	if len(orderFragmentIDBytes) != 32 {
		return orderFragmentID, ErrInvalidOrderFragmentIDLength
	}
	copy(orderFragmentID[:], orderFragmentIDBytes)
	return orderFragmentID, nil
}

func UnmarshalEncryptedCoExpShare(valueIn []string) (order.EncryptedCoExpShare, error) {
	var err error
	value := order.EncryptedCoExpShare{}
	if len(valueIn) != 2 {
		return value, ErrInvalidEncryptedCoExpShareLength
	}
	value.Co, err = base64.StdEncoding.DecodeString(valueIn[0])
	if err != nil {
		return value, err
	}
	value.Exp, err = base64.StdEncoding.DecodeString(valueIn[1])
	if err != nil {
		return value, err
	}
	return value, nil
}

func UnmarshalOrderFragment(orderFragmentIn OrderFragment) (ingress.OrderFragment, error) {
	var err error
	orderFragment := ingress.OrderFragment{EncryptedFragment: order.EncryptedFragment{}}
	orderFragment.Index = orderFragmentIn.Index
	orderFragment.EncryptedFragment.ID, err = UnmarshalOrderFragmentID(orderFragmentIn.ID)
	orderFragment.EncryptedFragment.EpochDepth = order.FragmentEpochDepth(orderFragmentIn.EpochDepth)
	if err != nil {
		return orderFragment, err
	}
	orderFragment.OrderID, err = UnmarshalOrderID(orderFragmentIn.OrderID)
	if err != nil {
		return orderFragment, err
	}
	orderFragment.OrderType = orderFragmentIn.OrderType
	orderFragment.OrderParity = orderFragmentIn.OrderParity
	orderFragment.OrderSettlement = orderFragmentIn.OrderSettlement
	orderFragment.OrderExpiry = time.Unix(orderFragmentIn.OrderExpiry, 0)
	orderFragment.Tokens, err = base64.StdEncoding.DecodeString(orderFragmentIn.Tokens)
	if err != nil {
		return orderFragment, err
	}
	orderFragment.Price, err = UnmarshalEncryptedCoExpShare(orderFragmentIn.Price)
	if err != nil {
		return orderFragment, err
	}
	orderFragment.Volume, err = UnmarshalEncryptedCoExpShare(orderFragmentIn.Volume)
	if err != nil {
		return orderFragment, err
	}
	orderFragment.MinimumVolume, err = UnmarshalEncryptedCoExpShare(orderFragmentIn.MinimumVolume)
	if err != nil {
		return orderFragment, err
	}
	orderFragment.Nonce, err = base64.StdEncoding.DecodeString(orderFragmentIn.Nonce)
	return orderFragment, nil
}

func UnmarshalOrderFragmentMapping(orderFragmentMappingIn OrderFragmentMapping) (order.ID, ingress.OrderFragmentMapping, error) {
	orderID := order.ID{}
	orderFragmentMapping := ingress.OrderFragmentMapping{}

	// Decode order ID
	for _, values := range orderFragmentMappingIn {
		var err error
		foundOrderID := false
		for _, value := range values {
			if orderID, err = UnmarshalOrderID(value.OrderID); err != nil {
				return orderID, orderFragmentMapping, err
			}
			foundOrderID = true
			break
		}
		if foundOrderID {
			break
		}
	}

	// Decode order fragments
	for key, orderFragmentsIn := range orderFragmentMappingIn {
		hashBytes, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			return orderID, orderFragmentMapping, fmt.Errorf("cannot decode pool hash %v: %v", key, err)
		}
		hash := [32]byte{}
		if len(hashBytes) != 32 {
			return orderID, orderFragmentMapping, ErrInvalidPodHashLength
		}
		copy(hash[:], hashBytes)
		orderFragmentMapping[hash] = make([]ingress.OrderFragment, 0, len(orderFragmentsIn))
		for _, orderFragmentIn := range orderFragmentsIn {
			orderFragment, err := UnmarshalOrderFragment(orderFragmentIn)
			if err != nil {
				return orderID, orderFragmentMapping, err
			}
			orderFragmentMapping[hash] = append(orderFragmentMapping[hash], orderFragment)
		}
	}
	return orderID, orderFragmentMapping, nil
}

func UnmarshalOrderFragmentMappings(orderFragmentMappingsIn OrderFragmentMappings) (order.ID, ingress.OrderFragmentMappings, error) {
	if len(orderFragmentMappingsIn) == 0 {
		return order.ID{}, ingress.OrderFragmentMappings{}, ErrEmptyOrderFragmentMapping
	}

	orderFragmentMappings := ingress.OrderFragmentMappings{}

	var orderID *order.ID
	for _, orderFragmentMappingIn := range orderFragmentMappingsIn {
		ordID, orderFragmentMapping, err := UnmarshalOrderFragmentMapping(orderFragmentMappingIn)
		if err != nil {
			return order.ID{}, ingress.OrderFragmentMappings{}, err
		}
		if orderID == nil {
			orderID = &ordID
		} else if !bytes.Equal(ordID[:], (*orderID)[:]) {
			return order.ID{}, ingress.OrderFragmentMappings{}, ErrInvalidOrderID
		}
		orderFragmentMappings = append(orderFragmentMappings, orderFragmentMapping)
	}

	return *orderID, orderFragmentMappings, nil
}
