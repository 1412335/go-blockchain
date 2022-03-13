package database

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
)

func getDatabaseDirPath(dataDir string) string {
	return path.Join(dataDir, "database")
}

func getGenesisJSONFilePath(dataDir string) string {
	return path.Join(getDatabaseDirPath(dataDir), "genesis.json")
}

func getBlocksDBFilePath(dataDir string) string {
	return path.Join(getDatabaseDirPath(dataDir), "blocks.db")
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

func initDataDirIfNotExists(dir string) error {
	if fileExists(getGenesisJSONFilePath(dir)) {
		return nil
	}

	if err := os.MkdirAll(getDatabaseDirPath(dir), os.ModePerm); err != nil {
		return err
	}

	gen := getGenesisJSONFilePath(dir)
	if err := writeGenesisToDisk(gen); err != nil {
		return err
	}

	if err := writeEmptyBlocksDBToDisk(getBlocksDBFilePath(dir)); err != nil {
		return err
	}

	return nil
}

func writeEmptyBlocksDBToDisk(dbPath string) error {
	return ioutil.WriteFile(dbPath, []byte(""), os.ModePerm)
}
