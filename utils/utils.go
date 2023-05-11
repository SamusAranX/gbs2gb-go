package utils

import (
	"io"
	"log"
	"os"
)

func ReadAllBytes(name string) ([]byte, error) {
	file, err := os.Open(name)
	if err != nil {
		log.Printf("Can't open file %s", name)
		return nil, err
	}

	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Can't read file %s", name)
		return nil, err
	}

	return fileBytes, nil
}

func WriteAllBytes(name string, data []byte) (int, error) {
	file, err := os.Create(name)
	if err != nil {
		return 0, err
	}

	defer file.Close()

	written, err := file.Write(data)
	if err != nil {
		return 0, err
	}

	return written, nil
}
