package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/nveeser/corepxe/merge/jsonwalk"
	"github.com/vincent-petithory/dataurl"
	"io"
	"log"
	"os"
)

var fileName = flag.String("filename", "", "filename to read")

func main() {
	flag.Parse()
	var f = os.Stdin
	if *fileName != "" {
		fmt.Printf("Open file\n")
		var err error
		f, err = os.OpenFile(*fileName, os.O_RDONLY, 000)
		if err != nil {
			log.Fatal(err)
		}
	}
	data, err := io.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	data, err = processJSON(data)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\n---\n%s\n---\n", data)
}

func processJSON(d []byte) ([]byte, error) {
	root := make(map[string]any)
	if err := json.Unmarshal(d, &root); err != nil {
		return nil, err
	}

	var opts = []jsonwalk.MergeOption{
		jsonwalk.IgnorePath("ignition"),
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
