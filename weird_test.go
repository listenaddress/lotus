package types

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
)

func blsaddr(n int64) address.Address {
	buf := make([]byte, 48)
	r := rand.New(rand.NewSource(n))
	r.Read(buf)

	addr, err := address.NewBLSAddress(buf)
	if err != nil {
		panic(err) // ok
	}

	return addr
}

func secp256k1Addr(n int64) address.Address {
	buf := make([]byte, 48)
	r := rand.New(rand.NewSource(n))
	r.Read(buf)

	addr, err := address.NewSecp256k1Address(buf)
	if err != nil {
		panic(err) // ok
	}

	return addr
}

func IDAddr(n uint64) address.Address {
	addr, err := address.NewIDAddress(n)
	if err != nil {
		panic(err) // ok
	}

	return addr
}

func ActorAddr(n int64) address.Address {
	buf := make([]byte, 48)
	r := rand.New(rand.NewSource(n))
	r.Read(buf)

	addr, err := address.NewActorAddress(buf)

	if err != nil {
		panic(err) // ok
	}

	return addr
}

func TestSerialize(t *testing.T) {
	var add address.Address
	add = IDAddr(13389068808970846986)
	fmt.Printf("\nadd %v", add)
	fmt.Printf("\nstring %v", add.String())

	var from address.Address
	from, _ = address.NewFromString("t2ch7krq7l35i74rebqbjdsp3ucl47t24e3juxjfa")
	fmt.Printf("\nfrom %v", from)
	fmt.Printf("\nbytes %v", from.Bytes())

	// otheradd, _ := address.NewFromString("t15ihq5ibzwki2b4ep2f46avlkrqzhpqgtga7pdrq")
	// fmt.Printf("\n otheradd string %v", otheradd.Bytes())

	m := &types.Message{
		To:     add,
		From:   from,
		Nonce:  11658068477141177407,
		Value:  types.NewInt(11416382733294334924),
		Method: 11471309754226496341,
		// Params:   []byte("some bytes, idk. probably at least ten of them"),
		GasLimit: types.NewInt(5210983352187082620),
		GasPrice: types.NewInt(7284457589948619162),
	}

	// fmt.Printf("m.From %v", m.Params)
	serialized, _ := m.Serialize()
	fmt.Printf("\nserialized %v", serialized)
}

func BenchmarkSerializeMessage(b *testing.B) {
	m := &types.Message{
		To:       blsaddr(1),
		From:     blsaddr(2),
		Nonce:    11658068477141177407,
		Method:   1231254,
		Params:   []byte("some bytes, idk. probably at least ten of them"),
		GasLimit: types.NewInt(126723),
		GasPrice: types.NewInt(1776234),
	}

	// fmt.Printf("m % v", m)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := m.Serialize()
		if err != nil {
			b.Fatal(err)
		}
	}
}
