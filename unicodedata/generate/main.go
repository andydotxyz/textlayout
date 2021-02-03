// Generate lookup function for Unicode properties not
// covered by the standard package unicode.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
)

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func goFormat(filename string) error {
	return exec.Command("goimports", "-w", filename).Run()
}

func main() {
	fetch := flag.Bool("download", false, "download the datas and save them locally (required at first usage)")
	flag.Parse()

	if *fetch {
		fetchData(urlXML)
		fetchData(urlUnicode)
		fetchData(urlEmoji)
		fetchData(urlMirroring)
		fetchData(urlArabic)
		// fetchData(urlBlocks)
	}

	processUnicode()
	processEmojis()
	processMirroring()
	processDecomposition() // use combiningClasses
	processArabicShaping()

	fmt.Println("Done.")
}

func processUnicode() {
	b, err := ioutil.ReadFile("UnicodeData.txt")
	check(err)

	err = parseUnicodeDatabase(b)
	check(err)

	fileName := "../combining_classes.go"
	file, err := os.Create(fileName)
	check(err)

	generateCombiningClasses(combiningClasses, file)

	err = file.Close()
	check(err)

	err = goFormat(fileName)
	check(err)
}

func processEmojis() {
	b, err := ioutil.ReadFile("emoji-data.txt")
	check(err)

	tables, err := parseAnnexTables(b)
	check(err)

	fileName := "../emojis.go"
	file, err := os.Create(fileName)
	check(err)

	generateEmojis(tables, file)

	err = file.Close()
	check(err)

	err = goFormat(fileName)
	check(err)
}

func processMirroring() {
	b, err := ioutil.ReadFile("BidiMirroring.txt")
	check(err)

	mirrors, err := parseMirroring(b)
	check(err)

	fileName := "../mirroring.go"
	file, err := os.Create(fileName)
	check(err)

	generateMirroring(mirrors, file)

	err = file.Close()
	check(err)

	err = goFormat(fileName)
	check(err)
}

func processDecomposition() {
	dms, compEx := parseXML("ucd.nounihan.grouped.zip")

	fileName := "../decomposition.go"
	file, err := os.Create(fileName)
	check(err)

	generateDecomposition(dms, compEx, file)

	err = file.Close()
	check(err)

	err = goFormat(fileName)
	check(err)
}

func processArabicShaping() {
	b, err := ioutil.ReadFile("ArabicShaping.txt")
	check(err)

	joiningTypes := parseArabicShaping(b)

	fileName := "../arabic.go"
	file, err := os.Create(fileName)
	check(err)

	generateArabicShaping(joiningTypes, file)

	err = file.Close()
	check(err)

	err = goFormat(fileName)
	check(err)
}