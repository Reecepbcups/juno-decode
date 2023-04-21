package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"runtime"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type Decode struct {
	// ID is the unique ID for the SQL database transaction
	ID int `json:"id"`
	// tx is the base64 amino in the input file, and the Decoded JSON in the output file
	Tx string `json:"tx"`
}

type Decodes []Decode

func decodeTx(clientCtx client.Context, wg *sync.WaitGroup, jobs <-chan Decode, results chan<- Decode) {
	defer wg.Done()
	for value := range jobs {
		txBytes, err := base64.StdEncoding.DecodeString(value.Tx)
		if err != nil {
			panic(err)
		}

		tx, err := clientCtx.TxConfig.TxDecoder()(txBytes)
		if err != nil {
			panic(err)
		}

		json, err := clientCtx.TxConfig.TxJSONEncoder()(tx)
		if err != nil {
			panic(err)
		}

		results <- Decode{
			ID: value.ID,
			Tx: string(json),
		}
	}
}

// GetDecodeCommand returns the decode command to take serialized bytes and turn
// it into a JSON-encoded transaction.
func GetFileDecodeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decode-file [file] [output-file-name]",
		Short: "Decode a bunch of amino bytre strings in 1 file. Then export",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			start := time.Now()
			// args := os.Args[1:]
			if len(args) < 2 {
				fmt.Println("Usage: ./juno-decoder tx decode-file input.json output.json")
				return
			}

			dat, err := ioutil.ReadFile(args[0])
			check(err)

			var values Decodes
			err = json.Unmarshal(dat, &values)
			check(err)

			clientCtx := client.GetClientContextFromCmd(cmd)

			jobs := make(chan Decode, len(values))
			results := make(chan Decode, len(values))

			var wg sync.WaitGroup

			cores := runtime.NumCPU()
			wg.Add(cores)
			for i := 0; i < cores; i++ {
				go decodeTx(clientCtx, &wg, jobs, results)
			}

			for _, value := range values {
				jobs <- value
			}
			close(jobs)

			newValues := make([]Decode, 0, len(values))
			for i := 0; i < len(values); i++ {
				newValues = append(newValues, <-results)
			}

			wg.Wait()

			output, err := json.Marshal(newValues)
			check(err)
			err = ioutil.WriteFile(args[1], output, 0644)
			check(err)

			fmt.Println("Decode time taken:", time.Since(start))

			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
