package tx

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	pb "github.com/xuperchain/xupercore/bcs/ledger/xledger/xldgpb"
	"github.com/xuperchain/xupercore/kernel/mock"
	"github.com/xuperchain/xupercore/lib/logs"
	"github.com/xuperchain/xupercore/protos"
)

var sum = 1000

func TestPutOrphanTx(t *testing.T) {
	econf, err := mock.NewEnvConfForTest()
	if err != nil {
		t.Fatal(err)
	}
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))
	l, _ := logs.NewLogger("1111", "test")
	isTest = true
	m := NewMempool(nil, l)
	id := "orphanTest"
	input := []*protos.TxInput{{RefTxid: []byte("orphanTest1")}}
	output := []*protos.TxOutput{{Amount: []byte("1")}}
	inputsExt := []*protos.TxInputExt{{RefTxid: []byte("orphanTest1")}}
	outputsExt := []*protos.TxOutputExt{{Bucket: "nil", Key: []byte("nil"), Value: []byte("nil")}}
	tx := NewTxForTest([]byte(id), input, output, inputsExt, outputsExt)
	e := m.PutTx(tx) // 添加孤儿交易，mempool 应该生成一个 mock node 以及把当前交易加入孤儿列表。
	if e != nil {
		t.Fatal(err)
	}
	if len(m.orphans) != 2 {
		t.Fatal("test failed for TestPutOrphanTx")
	}
	// printMempool(m)

	id1 := "orphanTest1"
	input1 := []*protos.TxInput{{RefTxid: []byte("orphanTest2")}}
	output1 := []*protos.TxOutput{{Amount: []byte("1")}}
	inputsExt1 := []*protos.TxInputExt{{RefTxid: []byte("orphanTest2")}}
	outputsExt1 := []*protos.TxOutputExt{{Bucket: "nil", Key: []byte("nil"), Value: []byte("nil")}}
	tx1 := NewTxForTest([]byte(id1), input1, output1, inputsExt1, outputsExt1)
	e = m.PutTx(tx1) // 添加上一个孤儿交易的所依赖的交易，也就是已经在 mempool 中的 mock node。orphans 中应该有三个节点。
	if e != nil {
		t.Fatal(err)
	}
	if len(m.orphans) != 3 {
		t.Fatal("test failed for TestPutOrphanTx when put mock node tx")
	}
	// printMempool(m)
}

func NewTxForTest(txid []byte, txInputs []*protos.TxInput, txOutput []*protos.TxOutput,
	txInputsExt []*protos.TxInputExt, txOutputsExt []*protos.TxOutputExt) *pb.Transaction {
	return &pb.Transaction{
		Txid:         txid,
		TxInputs:     txInputs,
		TxOutputs:    txOutput,
		TxInputsExt:  txInputsExt,
		TxOutputsExt: txOutputsExt,
	}
}

func TestConfirmTx(t *testing.T) {
	econf, err := mock.NewEnvConfForTest()
	if err != nil {
		t.Fatal(err)
	}
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))
	l, _ := logs.NewLogger("1111", "test")
	isTest = true
	m := NewMempool(nil, l)
	id := "orphanTest"
	input := []*protos.TxInput{{RefTxid: []byte("orphanTest1")}}
	output := []*protos.TxOutput{{Amount: []byte("1")}}
	inputsExt := []*protos.TxInputExt{{RefTxid: []byte("orphanTest1")}}
	outputsExt := []*protos.TxOutputExt{{Bucket: "nil", Key: []byte("nil"), Value: []byte("nil")}}
	tx := NewTxForTest([]byte(id), input, output, inputsExt, outputsExt)
	e := m.PutTx(tx) // 添加孤儿交易，mempool 应该生成一个 mock node 以及把当前交易加入孤儿列表。
	if e != nil {
		t.Fatal(err)
	}
	if len(m.orphans) != 2 {
		t.Fatal("test failed for TestConfirmTx")
	}

	m.ConfirmTxID(id)
	printMempool(m)
	if len(m.confirmed) != 0 || len(m.unconfirmed) != 0 || len(m.orphans) != 0 {
		t.Fatal("test failed for TestConfirmTx mempool size error")
	}
}

func TestMy(t *testing.T) {
	run(nil, t)
}

// 打包50010个交易，耗时200ms左右
func BenchmarkMempoolGetBatch(b *testing.B) {
	run(b, nil)
}

func printMempool(m *Mempool) {
	fmt.Println("MEMPOOL")
	fmt.Println("MEMPOOL unconfirmed len:", len(m.unconfirmed))
	fmt.Println("MEMPOOL confirmed len:", len(m.confirmed))
	fmt.Println("MEMPOOL orphans len:", len(m.orphans))
	fmt.Println("MEMPOOL bucketKeys len:", len(m.bucketKeyNodes))
}

func run(b *testing.B, t *testing.T) {
	if b != nil {
		sum = b.N
	}
	econf, err := mock.NewEnvConfForTest()
	if err != nil {
		t.Fatal(err)
	}
	logs.InitLog(econf.GenConfFilePath(econf.LogConf), econf.GenDirAbsPath(econf.LogDir))
	l, _ := logs.NewLogger("1111", "test")
	m := NewMempool(nil, l)
	setup(m)
	// printMempool(m)
	// return
	// now := time.Now()

	if b != nil {
		b.ResetTimer()
	}

	result := batchTx(m)
	printMempool(m)
	fmt.Println("确认一笔交易")
	cb := time.Now()
	// for i := 0; i < 10000; i++ {
	// 	e := m.ConfirmTx(result[i]) //
	// 	if e != nil {
	// 		panic(e)
	// 	}
	// }
	m.BatchConfirmTx(result[:10000])

	fmt.Println("confirm tx:", string(result[40].Txid), "耗时=", time.Since(cb))

	// deleteID := string(result[80].Txid) //"8001"
	// fmt.Println("delete tx:", deleteID)
	// m.DeleteTxAndChildren(deleteID)
	// m.ConfirmeTx(result[800])
	// if e != nil {
	// 	panic(e)
	// }
	printMempool(m)
	// _, ok := m.unconfirmed[deleteID]
	// fmt.Println("删除的交易在 未确认交易表吗？", ok)
	// fmt.Println("确认一笔交易OK")
	fmt.Println("再次打包")
	batchTx(m)
}

// var result []*pb.Transaction

// func ff(tx *pb.Transaction) bool {
// 	result = append(result, tx)
// 	return true
// }

func batchTx(m *Mempool) []*pb.Transaction {
	now := time.Now()
	var rrr []*pb.Transaction
	m.Range(func(tx *pb.Transaction) bool {
		rrr = append(rrr, tx)
		return true
	})
	end := time.Now()

	fmt.Println("耗时：", end.Sub(now))
	fmt.Println("打包交易量：", len(rrr))
	txids := make([]string, 0, len(rrr))
	for _, v := range rrr {
		txids = append(txids, string(v.Txid))
	}
	// fmt.Println(txids)
	return rrr
}

func setup(m *Mempool) {
	isTest = true
	type dbtxs struct {
		Txid string
	}

	txs := []string{
		"root0",
		"root1",
		"root2",
		"root3",
		"root4",
		"root5",
		"root6",
		"root7",
		"root8",
		"root9",
	}

	for _, t := range txs {
		id := []byte(t)
		tx0 := &pb.Transaction{
			Txid: id,
			TxInputs: []*protos.TxInput{
				{
					RefTxid: []byte("nil"),
				},
			},
			TxOutputs: []*protos.TxOutput{
				{
					Amount: []byte("1"),
				},
			},
			TxInputsExt: []*protos.TxInputExt{
				{
					RefTxid: []byte("nil"),
				},
			},
			TxOutputsExt: []*protos.TxOutputExt{
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
			},
		}
		dbTxs[string(id)] = tx0
	}

	for k, t := range txs {
		fatherID := []byte(t)
		id := strconv.Itoa(k)
		tx := &pb.Transaction{
			Txid: []byte(id),
			TxInputs: []*protos.TxInput{
				{
					RefTxid: fatherID,
				},
			},
			TxOutputs: []*protos.TxOutput{
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
				{
					Amount: []byte("1"),
				},
			},
			TxInputsExt: []*protos.TxInputExt{
				{
					RefTxid: fatherID,
				},
			},
			TxOutputsExt: []*protos.TxOutputExt{
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
				{
					Bucket: "nil",
					Key:    []byte("nil"),
					Value:  []byte("nil"),
				},
			},
		}
		m.PutTx(tx)
	}

	for ii := 1; ii <= sum; ii++ {
		for i := 0; i < 10; i++ {
			id := strconv.Itoa(ii*100 + i)
			tx1 := &pb.Transaction{
				Txid: []byte(id),
				TxInputs: []*protos.TxInput{
					{
						RefTxid:   []byte(strconv.Itoa(0 + (ii-1)*100)),
						RefOffset: int32(i), //int32(i)
					},
					{
						RefTxid:   []byte(strconv.Itoa(1 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(2 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(3 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(4 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(5 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(6 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(7 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(8 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(9 + (ii-1)*100)),
						RefOffset: int32(i),
					},
				},
				TxOutputs: []*protos.TxOutput{
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
					{
						Amount: []byte("1"),
					},
				},
				TxInputsExt: []*protos.TxInputExt{
					{
						RefTxid:   []byte(strconv.Itoa(0 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(1 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(2 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(3 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(4 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(5 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(6 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(7 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(8 + (ii-1)*100)),
						RefOffset: int32(i),
					},
					{
						RefTxid:   []byte(strconv.Itoa(9 + (ii-1)*100)),
						RefOffset: int32(i),
					},
				},
				TxOutputsExt: []*protos.TxOutputExt{
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
					{
						Bucket: "nil",
						Key:    []byte("nil"),
						Value:  []byte("nil"),
					},
				},
			}
			m.PutTx(tx1)
		}
	}
}
