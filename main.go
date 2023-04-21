package main

import (
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/client"

	codec "github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/spf13/cobra"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	// Latest app version here
	app "github.com/CosmosContracts/juno/v13/app"
	// older params here
	v12params "github.com/CosmosContracts/juno/v12/app/params"
	v13params "github.com/CosmosContracts/juno/v13/app/params"
)

// define the methods that are common to both v12 and v13 encoding configs
type EncodingConfig struct {
	InterfaceRegistry cdctypes.InterfaceRegistry
	Marshaler         codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

// V12
func V12MakeEncodingConfig() EncodingConfig {
	v := v12params.MakeEncodingConfig()
	return EncodingConfig{
		InterfaceRegistry: v.InterfaceRegistry,
		Marshaler:         v.Marshaler,
		TxConfig:          v.TxConfig,
		Amino:             v.Amino,
	}
}

// v13
func V13MakeEncodingConfig() EncodingConfig {
	v := v13params.MakeEncodingConfig()
	return EncodingConfig{
		InterfaceRegistry: v.InterfaceRegistry,
		Marshaler:         v.Marshaler,
		TxConfig:          v.TxConfig,
		Amino:             v.Amino,
	}
}

func MakeEncodingConfig(version string) EncodingConfig {
	if version == "v12" {
		return V12MakeEncodingConfig()
	} else if version == "v13" {
		return V13MakeEncodingConfig()
	} else {
		panic("invalid make encoding config version not found")
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./juno-decoder v13 tx decode <tx-b64>")
		return
	}

	// maybe put this at the very end?
	appVersion := ""
	switch os.Args[1] {
	case "v13":
		appVersion = "v13"
	case "v12":
		appVersion = "v12"
	default:
		panic("Usage: ./juno-decoder v13 ...")
	}

	os.Args = append(os.Args[:1], os.Args[2:]...)
	fmt.Println("Updated os.Args: ", os.Args)

	rootCmd, _ := NewRootCmd(appVersion)
	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}

func NewRootCmd(appVersion string) (*cobra.Command, EncodingConfig) {
	encodingConfig := MakeEncodingConfig(appVersion)

	// print out each of encodingConfig
	// fmt.Println("InterfaceRegistry: ", encodingConfig.InterfaceRegistry)
	// fmt.Println("Marshaler: ", encodingConfig.Marshaler)
	// fmt.Println("TxConfig: ", encodingConfig.TxConfig)
	// fmt.Println("Amino: ", encodingConfig.Amino)

	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	cfg.SetBech32PrefixForValidator(app.Bech32PrefixValAddr, app.Bech32PrefixValPub)
	cfg.SetBech32PrefixForConsensusNode(app.Bech32PrefixConsAddr, app.Bech32PrefixConsPub)
	cfg.Seal()

	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Marshaler).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		// WithBroadcastMode(flags.BroadcastBlock).
		WithHomeDir(app.DefaultNodeHome).
		WithViper("")

	rootCmd := &cobra.Command{
		Use:   version.AppName,
		Short: "Juno Network Lightweight Decoder",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			return server.InterceptConfigsPreRunHandler(cmd, "", nil)
		},
	}

	initRootCmd(rootCmd, encodingConfig)

	return rootCmd, encodingConfig
}

func initRootCmd(rootCmd *cobra.Command, encodingConfig EncodingConfig) {
	rootCmd.AddCommand(
		txCommand(),
	)
}

func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetDecodeCommand(),
		GetFileDecodeCommand(),
	)

	return cmd
}
