package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/c-bata/go-prompt"
)

type resourceOptions struct {
	Namespace string
	Cluster   string
	Shard     int
	Replica   int
	Nodes     []string
	Type      string
}

func parseOptions(args []string, allowEmptyValue bool) (*resourceOptions, error) {
	options := &resourceOptions{Shard: -1}
	for i := 0; i < len(args); i++ {
		lastArg := i == len(args)-1
		switch strings.ToLower(args[i]) {
		case "--namespace":
			if lastArg {
				return nil, errors.New("missing namespace value")
			}
			i++
			if strings.HasPrefix(args[i], "-") {
				return nil, errors.New("namespace should NOT start with '-'")
			}
			options.Namespace = args[i]
		case "--cluster":
			if lastArg {
				if allowEmptyValue {
					return options, nil
				}
				return nil, errors.New("missing cluster value")
			}
			i++
			if strings.HasPrefix(args[i], "-") {
				return nil, errors.New("cluster should NOT start with '-'")
			}
			options.Cluster = args[i]
		case "--shard":
			if lastArg {
				if allowEmptyValue {
					return options, nil
				}
				return options, errors.New("missing shard value")
			}
			i++
			shard, err := strconv.Atoi(args[i])
			if err != nil {
				return nil, fmt.Errorf("parse shard: %w", err)
			}
			if shard < 0 {
				return nil, fmt.Errorf("shard should be >= 0")
			}
			options.Shard = shard
		case "--replica":
			if lastArg {
				if allowEmptyValue {
					return options, nil
				}
				return nil, errors.New("missing replica value")
			}
			i++
			replica, err := strconv.Atoi(args[i])
			if err != nil {
				return nil, fmt.Errorf("parse replica: %w", err)
			}
			if replica <= 0 {
				return nil, fmt.Errorf("replica should be > 0")
			}
			options.Replica = replica
		case "--nodes":
			if lastArg {
				if allowEmptyValue {
					return options, nil
				}
				return nil, errors.New("missing replica value")
			}
			i++
			if strings.HasPrefix(args[i], "-") {
				return nil, errors.New("nodes should NOT start with '-'")
			}
			nodes := strings.Split(strings.TrimSpace(args[i]), ",")
			options.Nodes = make([]string, 0)
			for _, node := range nodes {
				if len(node) == 0 {
					continue
				}
				options.Nodes = append(options.Nodes, node)
			}
		case "--type":
			if lastArg {
				if allowEmptyValue {
					return options, nil
				}
				return nil, errors.New("missing replica value")
			}
			i++
			typ := strings.ToLower(args[0])
			if typ != "pending" && typ != "history" {
				return nil, errors.New("--type must be 'pending' or 'history'")
			}
			options.Type = typ
		default:
			if !strings.HasPrefix(args[i], "--") {
				continue
			}
			return nil, fmt.Errorf("unknown option '%s'", args[i])
		}
	}
	return options, nil
}

func (c *Completer) optionCompleter(args []string, _ string) []prompt.Suggest {
	options := []prompt.Suggest{
		{Text: "--namespace"},
		{Text: "--cluster"},
		{Text: "--shard"},
	}
	if strings.ToLower(args[0]) == operationCreate && strings.ToLower(args[1]) == resourceCluster {
		options = append(options, []prompt.Suggest{
			{Text: "--nodes"},
			{Text: "--replica"},
		}...)
	}
	return options
}
