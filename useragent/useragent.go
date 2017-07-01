package useragent

import (
	"bufio"
	"log"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"
)

var (
	_, b, _, _ = runtime.Caller(0)
	baseDir    = filepath.Dir(b)
	userAgents = []string{}
)

func init() {

	file, err := os.Open(path.Join(baseDir, "user-agents.txt"))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		userAgents = append(userAgents, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func GetRandom() string {
	rand.Seed(time.Now().UnixNano())
	n := rand.Int() % len(userAgents)
	return userAgents[n]
}
