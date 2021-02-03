package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"unicode"

	"github.com/benoitkugler/textlayout/unicodedata"
	"golang.org/x/text/unicode/rangetable"
)

const header = `
package unicodedata

// Code generated by generate/main.go DO NOT EDIT.

`

func sortRunes(rs []rune) {
	sort.Slice(rs, func(i, j int) bool { return rs[i] < rs[j] })
}

// compacts the code more than a "%#v" directive
func printTable(rt *unicode.RangeTable, omitTypeLitteral bool) string {
	w := new(bytes.Buffer)
	if omitTypeLitteral {
		fmt.Fprintln(w, "{")
	} else {
		fmt.Fprintln(w, "&unicode.RangeTable{")
	}
	if len(rt.R16) > 0 {
		fmt.Fprintln(w, "\tR16: []unicode.Range16{")
		for _, r := range rt.R16 {
			fmt.Fprintf(w, "\t\t{Lo:%#04x, Hi:%#04x, Stride:%d},\n", r.Lo, r.Hi, r.Stride)
		}
		fmt.Fprintln(w, "\t},")
	}
	if len(rt.R32) > 0 {
		fmt.Fprintln(w, "\tR32: []unicode.Range32{")
		for _, r := range rt.R32 {
			fmt.Fprintf(w, "\t\t{Lo:%#x, Hi:%#x,Stride:%d},\n", r.Lo, r.Hi, r.Stride)
		}
		fmt.Fprintln(w, "\t},")
	}
	if rt.LatinOffset > 0 {
		fmt.Fprintf(w, "\tLatinOffset: %d,\n", rt.LatinOffset)
	}
	fmt.Fprintf(w, "}")
	return w.String()
}

func generateCombiningClasses(classes map[uint8][]rune, w io.Writer) {
	fmt.Fprintln(w, header)

	// create and compact the tables
	var out [256]*unicode.RangeTable
	for k, v := range classes {
		if len(v) == 0 {
			return
		}
		out[k] = rangetable.New(v...)
	}

	// print them
	fmt.Fprintln(w, "var combiningClasses = [256]*unicode.RangeTable{")
	for i, t := range out {
		if t == nil {
			continue
		}
		fmt.Fprintf(w, "%d : %s,\n", i, printTable(t, true))
	}
	fmt.Fprintln(w, "}")
}

func generateEmojis(runes map[string][]rune, w io.Writer) {
	fmt.Fprintln(w, header)
	var classes = [...]string{"Emoji", "Emoji_Presentation", "Emoji_Modifier", "Emoji_Modifier_Base", "Extended_Pictographic"}
	for _, class := range classes {
		table := rangetable.New(runes[class]...)
		s := printTable(table, false)
		fmt.Fprintf(w, "var %s = %s\n\n", class, s)
	}

}
func generateMirroring(runes map[uint16]uint16, w io.Writer) {
	fmt.Fprintln(w, header)
	fmt.Fprintf(w, "var mirroring = map[rune]rune{ // %d entries \n", len(runes))
	var sorted []rune
	for r1 := range runes {
		sorted = append(sorted, rune(r1))
	}
	sortRunes(sorted)
	for _, r1 := range sorted {
		r2 := runes[uint16(r1)]
		fmt.Fprintf(w, "0x%04x: 0x%04x,\n", r1, r2)
	}
	fmt.Fprintln(w, "}")
}

func generateDecomposition(dms map[rune][]rune, compExp map[rune]bool, w io.Writer) {
	var (
		decompose1 [][2]rune         // length 1 mappings {from, to}
		decompose2 [][3]rune         // length 2 mappings {from, to1, to2}
		compose    [][3]rune         // length 2 mappings {from1, from2, to}
		ccc        = map[rune]bool{} // has combining class
	)
	for c, runes := range combiningClasses {
		for _, r := range runes {
			ccc[r] = c != 0
		}
	}
	for r, v := range dms {
		switch len(v) {
		case 1:
			decompose1 = append(decompose1, [2]rune{r, v[0]})
		case 2:
			decompose2 = append(decompose2, [3]rune{r, v[0], v[1]})
			var composed rune
			if !compExp[r] && !ccc[r] {
				composed = r
			}
			compose = append(compose, [3]rune{v[0], v[1], composed})
		default:
			log.Fatalf("unexpected runes for decomposition: %d, %v", r, v)
		}
	}

	// sort for determinisme
	sort.Slice(decompose1, func(i, j int) bool { return decompose1[i][0] < decompose1[j][0] })
	sort.Slice(decompose2, func(i, j int) bool { return decompose2[i][0] < decompose2[j][0] })
	sort.Slice(compose, func(i, j int) bool {
		return compose[i][0] < compose[j][0] ||
			compose[i][0] == compose[j][0] && compose[i][1] < compose[j][1]
	})

	fmt.Fprintln(w, header)

	fmt.Fprintf(w, "var decompose1 = map[rune]rune{ // %d entries \n", len(decompose1))
	for _, vals := range decompose1 {
		fmt.Fprintf(w, "0x%04x: 0x%04x,\n", vals[0], vals[1])
	}
	fmt.Fprintln(w, "}")

	fmt.Fprintf(w, "var decompose2 = map[rune][2]rune{ // %d entries \n", len(decompose2))
	for _, vals := range decompose2 {
		fmt.Fprintf(w, "0x%04x: {0x%04x,0x%04x},\n", vals[0], vals[1], vals[2])
	}
	fmt.Fprintln(w, "}")

	fmt.Fprintf(w, "var compose = map[[2]rune]rune{ // %d entries \n", len(compose))
	for _, vals := range compose {
		fmt.Fprintf(w, "{0x%04x,0x%04x}: 0x%04x,\n", vals[0], vals[1], vals[2])
	}
	fmt.Fprintln(w, "}")
}

func generateArabicShaping(joining map[rune]unicodedata.ArabicJoining, w io.Writer) {
	fmt.Fprintln(w, header)

	// Joining

	// sort for determinism
	var keys []rune
	for r := range joining {
		keys = append(keys, r)
	}
	sortRunes(keys)

	fmt.Fprintf(w, "var ArabicJoinings = map[rune]ArabicJoining{ // %d entries \n", len(keys))
	for _, r := range keys {
		fmt.Fprintf(w, "0x%04x: %q,\n", r, joining[r])
	}
	fmt.Fprintln(w, "}")

	// Shaping

	if shapingTable.max < shapingTable.min {
		check(errors.New("error: no shaping pair found, something wrong with reading input"))
	}

	fmt.Fprintf(w, "const FirstArabicShape = 0x%04x\n", shapingTable.min)
	fmt.Fprintf(w, "const LastArabicShape = 0x%04x\n", shapingTable.max)

	fmt.Fprintln(w, `
	// ArabicShaping defines the shaping for arabic runes. Each entry is indexed by
	// the shape, between 0 and 3:
	//   - 0: isolated
	//   - 1: final
	//   - 2: initial
	//   - 3: medial
	// See also the bounds given by FirstArabicShape and LastArabicShape.`)
	fmt.Fprintf(w, "var ArabicShaping = [...][4]uint16{ // required memory: %d KB \n", (shapingTable.max-shapingTable.min+1)*4*4/1000)
	for c := shapingTable.min; c <= shapingTable.max; c++ {
		fmt.Fprintf(w, "{%d,%d,%d,%d},\n",
			// fmt.Fprintf(w, "{0x%04x,0x%04x,0x%04x,0x%04x},\n",
			shapingTable.table[c][0], shapingTable.table[c][1], shapingTable.table[c][2], shapingTable.table[c][3])
	}
	fmt.Fprintln(w, "}")

	// Ligatures

	ligas := map[rune][][2]rune{}
	for pair, shapes := range ligatures {
		for shape, c := range shapes {
			if c == 0 {
				continue
			}
			var liga [2]rune
			if shape == 0 {
				liga = [2]rune{shapingTable.table[pair[0]][2], shapingTable.table[pair[1]][1]}
			} else if shape == 1 {
				liga = [2]rune{shapingTable.table[pair[0]][3], shapingTable.table[pair[1]][1]}
			} else {
				check(fmt.Errorf("unexpected shape %d", shape))
			}
			ligas[liga[0]] = append(ligas[liga[0]], [2]rune{liga[1], c})
		}
	}
	var (
		maxI   int
		sorted []rune
	)
	for r, v := range ligas {
		if len(v) > maxI {
			maxI = len(v)
		}
		sorted = append(sorted, r)
	}
	sortRunes(sorted)

	fmt.Fprintln(w)
	fmt.Fprintf(w, `var ligatureTable = [...]struct{
	 	first uint16
		pairs [%d][2]uint16 // {second, ligature}
	} {`, maxI)
	fmt.Fprintln(w)
	for _, first := range sorted {
		fmt.Fprintf(w, "  { 0x%04x, [%d][2]uint16{\n", first, maxI)
		for _, liga := range ligas[first] {
			fmt.Fprintf(w, "    { 0x%04x, 0x%04x },\n", liga[0], liga[1])
		}
		fmt.Fprintln(w, "  }},")
	}
	fmt.Fprintln(w, "};")
	fmt.Fprintln(w)
}