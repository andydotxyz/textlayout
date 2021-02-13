package truetype

import (
	"os"
	"testing"
)

func TestGPOS(t *testing.T) {
	filenames := []string{
		"testdata/Raleway-v4020-Regular.otf",
		"testdata/Estedad-VF.ttf",
		"testdata/Mada-VF.ttf",
	}

	filenames = append(filenames, dirFiles(t, "testdata/layout_fonts/gpos")...)

	for _, filename := range filenames {
		file, err := os.Open(filename)
		if err != nil {
			t.Fatalf("Failed to open %q: %s\n", filename, err)
		}

		font, err := Parse(file)
		if err != nil {
			t.Fatalf("Parse(%q) err = %q, want nil", filename, err)
		}

		_, err = font.GposTable()
		if err != nil {
			t.Fatal(filename, err)
		}
		// for _, l := range sub.Lookups {
		// 	for _, s := range l.Subtables {
		// 		if s.Data == nil {
		// 			continue
		// 		}
		// 	}
		// }
		// fmt.Println(len(sub.Lookups), "lookups")
	}
}