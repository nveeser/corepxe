package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	jsonwalk "github.com/nveeser/srvsrv/jsonwalk"
	"github.com/vincent-petithory/dataurl"
	"io"
	"log"
	"os"
	"strings"
)

var input = flag.String("input", "", "filename to read")
var output = flag.String("output", "", "filename to write")
var ignores = flag.String("ignore_paths", "", "comma separate list of paths to ignore")
var replaces = flag.String("replace_paths", "", "comma separate list of paths to replace (vs append)")

func main() {
	flag.Parse()
	var in = os.Stdin
	if *input != "" {
		var err error
		in, err = os.OpenFile(*input, os.O_RDONLY, 000)
		if err != nil {
			log.Fatal(err)
		}
	}
	data, err := io.ReadAll(in)
	if err != nil {
		log.Fatal(err)
	}
	data, err = processJSON(data)
	if err != nil {
		log.Fatal(err)
	}
	var out = os.Stdout
	if *output != "" {
		var err error
		out, err = os.OpenFile(*output, os.O_WRONLY, 777)
		if err != nil {
			log.Fatal(err)
		}
	}
	if _, err := io.Copy(out, bytes.NewReader(data)); err != nil {
		log.Fatal(err)
	}
}

func processJSON(d []byte) ([]byte, error) {
	root := make(map[string]any)
	if err := json.Unmarshal(d, &root); err != nil {
		return nil, err
	}

	var opts []jsonwalk.MergeOption
	for _, path := range strings.Split(*ignores, ",") {
		opts = append(opts, jsonwalk.IgnorePath(path))
	}
	for _, path := range strings.Split(*replaces, ",") {
		opts = append(opts, jsonwalk.Replace(path))
	}

	for v := range jsonwalk.Find(root, "ignition.config.merge.*") {
		mergeMap := v.(map[string]any)
		jsonMap, err := decodeMerge(mergeMap)
		if err != nil {
			return nil, fmt.Errorf("dataurl: %w", err)
		}
		if err := jsonwalk.Merge(root, jsonMap, opts...); err != nil {
			return nil, fmt.Errorf("error Merge(): %w", err)
		}
	}
	for v := range jsonwalk.Find(root, "ignition.config") {
		configObj, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("ignition.config is type %T not %T", configObj, map[string]any{})
		}
		delete(configObj, "merge")
	}
	return json.MarshalIndent(root, "", " ")
}

func decodeMerge(encoded map[string]any) (map[string]any, error) {
	durl, err := dataurl.DecodeString(encoded["source"].(string))
	if err != nil {
		return nil, fmt.Errorf("dataurl: %w", err)
	}
	gr, err := gzip.NewReader(bytes.NewReader(durl.Data))
	if err != nil {
		return nil, fmt.Errorf("gzip.NewReader: %w", err)
	}
	mergeJSON, err := io.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	decoded := make(map[string]any)
	if err := json.Unmarshal(mergeJSON, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}
