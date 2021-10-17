package tx

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tm "github.com/tendermint/tendermint/types"
	tmdb "github.com/tendermint/tm-db"
	"github.com/terra-money/mantlemint-provider-v0.34.x/indexer"
	"github.com/terra-money/mantlemint-provider-v0.34.x/mantlemint"
	"io/ioutil"
	"net/http"
	"strconv"
)

func txByHashHandler(indexerDB tmdb.DB, txHash string) ([]byte, error) {
	return indexerDB.Get(getKey(txHash))
}

func txsByHeightHandler(indexerDB tmdb.DB, height string) ([]byte, error) {
	heightInInt, err := strconv.Atoi(height)
	if err != nil {
		return nil, fmt.Errorf("invalid height: %v", err)
	}
	return indexerDB.Get(getByHeightKey(uint64(heightInInt)))
}

var RegisterRESTRoute = indexer.CreateRESTRoute(func(router *mux.Router, postRouter *mux.Router, indexerDB tmdb.DB) {
	router.HandleFunc("/index/tx/by_hash/{hash}", func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)
		hash, ok := vars["hash"]
		if !ok {
			http.Error(writer, "txn not found", 400)
			return
		}

		if txn, err := txByHashHandler(indexerDB, hash); err != nil {
			http.Error(writer, err.Error(), 400)
			return
		} else {
			writer.WriteHeader(200)
			writer.Write(txn)
			return
		}
	}).Methods("GET")

	router.HandleFunc("/index/tx/by_height/{height}", func(writer http.ResponseWriter, request *http.Request) {
		vars := mux.Vars(request)
		height, ok := vars["height"]
		if !ok {
			http.Error(writer, "invalid height", 400)
			return
		}

		if txns, err := txsByHeightHandler(indexerDB, height); err != nil {
			http.Error(writer, err.Error(), 400)
			return
		} else if txns == nil {
			http.Error(writer, fmt.Errorf("invalid height; you may be requesting for a block not seen yet").Error(), 204)
			return
		} else {
			writer.WriteHeader(200)
			writer.Write(txns)
			return
		}
	}).Methods("GET")

	// expected input is from RPC
	// { block, txRecords }
	postRouter.HandleFunc("/index/txs", func(writer http.ResponseWriter, request *http.Request) {
		body, err := request.GetBody()
		if err != nil {
			http.Error(writer, err.Error(), 400)
			return
		}

		bz, err := ioutil.ReadAll(body)
		if err != nil {
			http.Error(writer, err.Error(), 400)
			return
		}

		input := struct {
			Block     *tm.Block  `json:"block"`
			TxRecords []TxRecord `json:"tx_records"`
		}{
			Block:     nil,
			TxRecords: make([]TxRecord, 0),
		}

		if err := tmjson.Unmarshal(bz, &input); err != nil {
			http.Error(writer, err.Error(), 400)
			return
		}

		batch := indexerDB.NewBatch()
		evc := mantlemint.NewMantlemintEventCollector()

		for _, txRecord := range input.TxRecords {
			deliverTx := abci.ResponseDeliverTx{}
			if err := tmjson.Unmarshal(txRecord.TxResponse, &deliverTx); err != nil {
				http.Error(writer, errors.Wrapf(err, "failed unmarshaling tx response").Error(), 400)
			}
			evc.ResponseDeliverTxs = append(evc.ResponseDeliverTxs, &deliverTx)
		}

		if err := IndexTx(batch, input.Block, nil, evc); err != nil {
			http.Error(writer, err.Error(), 400)
			return
		}

		if err := batch.WriteSync(); err != nil {
			http.Error(writer, err.Error(), 400)
			return
		}

		if err := batch.Close(); err != nil {
			http.Error(writer, err.Error(), 400)
			return
		}
	})
})
