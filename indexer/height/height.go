package height

import (
	"fmt"

	terra "github.com/crescent-network/crescent/app"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tm "github.com/tendermint/tendermint/types"
	"github.com/terra-money/mantlemint/db/safe_batch"
	"github.com/terra-money/mantlemint/indexer"
	"github.com/terra-money/mantlemint/mantlemint"
)

var IndexHeight = indexer.CreateIndexer(func(indexerDB safe_batch.SafeBatchDB, block *tm.Block, _ *tm.BlockID, _ *mantlemint.EventCollector, _ *terra.App) error {
	defer fmt.Printf("[indexer/height] indexing done for height %d\n", block.Height)
	height := block.Height

	record := HeightRecord{Height: uint64(height)}
	recordJSON, recordErr := tmjson.Marshal(record)
	if recordErr != nil {
		return recordErr
	}

	return indexerDB.Set(getKey(), recordJSON)
})
