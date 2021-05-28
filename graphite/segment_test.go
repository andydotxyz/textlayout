package graphite

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"

	"github.com/benoitkugler/textlayout/fonts/truetype"
)

// Test shaping output against the reference graphite implementation

type testOptions struct {
	input []rune
	// justification int // we dont support neither test this feature
	offset int // zero for us
}

func lookup(map_ []*Slot, val *Slot) int {
	i := 0
	for ; map_[i] != val && map_[i] != nil; i++ {
	}
	if map_[i] != nil {
		return i
	}
	return -1
}

func (opts testOptions) dumpSegment(seg *Segment) ([]byte, error) {
	// int i = 0;
	// float advanceWidth;
	// #ifndef NDEBUG
	// 	int numSlots = gr_seg_n_slots(seg);
	// #endif
	//        size_t *map = new size_t [seg.length() + 1];
	// if (opts.justification > 0){
	// 	advanceWidth = gr_seg_justify(seg, gr_seg_first_slot(seg), sizedFont, gr_seg_advance_X(seg) * opts.justification / 100., gr_justCompleteLine, NULL, NULL);
	// }else{
	advanceWidth := seg.Advance.X
	map_ := make([]*Slot, seg.NumGlyphs+1)
	for slot, i := seg.First, 0; slot != nil; slot, i = slot.Next, i+1 {
		map_[i] = slot
	}
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, "Segment length: %d\n", seg.NumGlyphs)
	fmt.Fprintf(buf, "pos  gid   attach\t     x\t     y\tins bw\t  chars\t\tUnicode\t")
	fmt.Fprintf(buf, "\n")
	i := 0
	for slot := seg.First; slot != nil; slot, i = slot.Next, i+1 {
		// consistency check for last slot
		assertion := ((i+1 < seg.NumGlyphs) || (slot == seg.last))
		if !assertion {
			return nil, fmt.Errorf("invalid slot index: %d %d", i, seg.NumGlyphs)
		}
		orgX := slot.Position.X
		orgY := slot.Position.Y
		cinfo := seg.getCharInfo(slot.original)
		breakWeight := 0
		if cinfo != nil {
			breakWeight = int(cinfo.breakWeight)
		}
		fmt.Fprintf(buf, "%02d  %4d %3d@%d,%d\t%6.1f\t%6.1f\t%2d%4d\t%3d %3d\t",
			i, slot.GlyphID, lookup(map_, slot.parent),
			slot.getAttr(seg, gr_slatAttX, 0), slot.getAttr(seg, gr_slatAttY, 0),
			orgX, orgY, boolToInt(slot.CanInsertBefore()),
			breakWeight, slot.Before, slot.After)

		if slot.Before+opts.offset < len(opts.input) && slot.After+opts.offset < len(opts.input) {
			fmt.Fprintf(buf, "%7x\t%7x",
				opts.input[slot.Before+opts.offset],
				opts.input[slot.After+opts.offset])
		}
		fmt.Fprintf(buf, "\n")
	}
	assertion := (i == seg.NumGlyphs)
	if !assertion {
		return nil, fmt.Errorf("wrong number of slots: %d != %d", i, seg.NumGlyphs)
	}
	// assign last point to specify advance of the whole array
	// position arrays must be one bigger than what countGlyphs() returned
	fmt.Fprintf(buf, "Advance width = %6.1f\n", advanceWidth)
	fmt.Fprintf(buf, "\nChar\tUnicode\tBefore\tAfter\tBase\n")
	for j, c := range seg.charinfo {
		fmt.Fprintf(buf, "%d\t%04X\t%d\t%d\t%d\n", j, c.char, c.before, c.after, c.base)
	}

	return buf.Bytes(), nil
}

var fonttestInput = []struct {
	name, fontfile string
	text           []rune
	features       string
	rtl            bool
}{
	{"padauk1", "Padauk.ttf", []rune{0x1015, 0x102F, 0x100F, 0x1039, 0x100F, 0x1031, 0x1038}, "", false},
	{"padauk2", "Padauk.ttf", []rune{0x1000, 0x103C, 0x102D, 0x102F}, "", false},
	{"padauk3", "Padauk.ttf", []rune{0x101e, 0x1004, 0x103a, 0x1039, 0x1001, 0x103b, 0x102d, 0x102f, 0x1004, 0x103a, 0x1038}, "", false},
	{"padauk4", "Padauk.ttf", []rune{0x1005, 0x1000, 0x1039, 0x1000, 0x1030}, "", false},
	{"padauk5", "Padauk.ttf", []rune{0x1000, 0x103c, 0x1031, 0x102c, 0x1004, 0x1037, 0x103a}, "", false},
	{"padauk6", "Padauk.ttf", []rune{0x1000, 0x102D, 0x1005, 0x1039, 0x1006, 0x102C}, "", false},
	// padauk7 can cause an infinite loop, though the text is miss-spelt
	{"padauk7", "Padauk.ttf", []rune{0x1017, 0x1014, 0x103c, 0x103d, 0x102f}, "", false},
	{"padauk8", "Padauk.ttf", []rune{0x1004, 0x103A, 0x1039, 0x1005}, "", false},
	{"padauk9", "Padauk.ttf", []rune{0x1004, 0x103A, 0x1039}, "", false},
	{"padauk10", "Padauk.ttf", []rune{0x1004, 0x103D, 0x1000, 0x103A}, "kdot=1,wtri=1", false},
	{"padauk11", "Padauk.ttf", []rune{0x100B, 0x1039, 0x100C, 0x1031, 0x102C}, "", false},
	// {"padauk12", "Padauk.ttf", []rune{0x0048, 0x0065, 0x006C,0x006C,0x006F,0x0020,0x004D,0x0075,0x006D -j 107}},
	{"scher1", "Scheherazadegr.ttf", []rune{0x0628, 0x0628, 0x064E, 0x0644, 0x064E, 0x0654, 0x0627, 0x064E}, "", true},
	{"scher2", "Scheherazadegr.ttf", []rune{0x0627, 0x0644, 0x0625, 0x0639, 0x0644, 0x0627, 0x0646}, "", true},
	{"scher3", "Scheherazadegr.ttf", []rune{0x0627, 0x0031, 0x0032, 0x002D, 0x0034, 0x0035, 0x0627}, "", true},
	{"scher4", "Scheherazadegr.ttf", []rune{0x0627, 0x0653, 0x06AF}, "", true},
	// {"scher5", "Scheherazadegr_noglyfs.ttf", []rune{0x0627, 0x0653, 0x06AF}, "", true},
	{"charis1", "charis.ttf", []rune{0x0069, 0x02E6, 0x02E8, 0x02E5}, "", false},
	{"charis2", "charis.ttf", []rune{0x1D510, 0x0041, 0x1D513}, "", false},
	{"charis3", "charis.ttf", []rune{0x0054, 0x0069, 0x1ec3, 0x0075}, "lang=vie", false},
	{"charis4", "charis.ttf", []rune{0x006b, 0x0361, 0x070}, "", false},
	{"charis5", "charis.ttf", []rune{0x0020, 0x006C, 0x0325, 0x0065}, "", false},
	// {"charis6", "charis.ttf", []rune{0x0048, 0x0065, 0x006C,0x006C,0x006F,0x0020,0x004D,0x0075,0x006D -j 107}, "", false},
	{"charis7", "charis_fast.ttf", []rune{0x0049, 0x0065, 0x006C, 0x006C, 0x006F}, "", false},
	{"charis8", "charis.ttf", []rune{0x0054, 0x0069, 0x1ec3, 0x007}, "lang=vi  ", false},
	{"magyar1", "MagyarLinLibertineG.ttf", []rune{0x0031, 0x0035}, "210=36", false},
	{"magyar2", "MagyarLinLibertineG.ttf", []rune{0x0031, 0x0030}, "210=200", false},
	{"magyar3", "MagyarLinLibertineG.ttf", []rune{0x0066, 0x0069, 0x0066, 0x0074, 0x0079, 0x002d, 0x0066, 0x0069, 0x0076, 0x0065}, "209=3", false},
	{"grtest1", "grtest1gr.ttf", []rune{0x0062, 0x0061, 0x0061, 0x0061, 0x0061, 0x0061, 0x0061, 0x0062, 0x0061}, "", false},
	{"general1", "general.ttf", []rune{0x0E01, 0x0062}, "", false},
	{"piglatin1", "PigLatinBenchmark_v3.ttf", []rune{0x0068, 0x0065, 0x006C, 0x006C, 0x006F}, "", false},
}

func parseFeatures(face *GraphiteFace, features string) (FeaturesValue, []byte, error) {
	if features == "" {
		return nil, nil, nil
	}

	// special case for language
	if strings.HasPrefix(features, "lang=") {
		var buf [4]byte
		copy(buf[:], features[5:])
		langID := truetype.MustNewTag(string(buf[:]))
		return face.FeaturesForLang(langID), nil, nil
	}

	var (
		out FeaturesValue
		buf = new(bytes.Buffer)
	)
	for _, feature := range strings.Split(features, ",") {
		fg := strings.Split(feature, "=")
		if len(fg) != 2 {
			return nil, nil, fmt.Errorf("invalid feature format: %s", feature)
		}
		val, err := strconv.Atoi(fg[1])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid feature format %s: %s", feature, err)
		}
		// feature id is either a 4 bytes-tag or a decimal digit
		featTag, err := strconv.Atoi(fg[0])
		if err != nil {
			if len(fg[0]) != 4 {
				return nil, nil, fmt.Errorf("invalid feature format: %s", feature)
			}
			featTag = int(truetype.MustNewTag(fg[0]))
		}
		tag := truetype.Tag(featTag)
		out = append(out, FeatureValue{Id: tag, Value: int16(val)})
		if featTag > 0x20000000 {
			fmt.Fprintf(buf, "%s=%d\n", tag.String(), val)
		} else {
			fmt.Fprintf(buf, "%d=%d\n", tag, val)
		}
	}
	return out, buf.Bytes(), nil
}

func checkSegmentNumGlyphs(seg *Segment) error {
	var nb int
	for s := seg.First; s != nil; s = s.Next {
		nb++
	}
	if nb != seg.NumGlyphs {
		return fmt.Errorf("invalid number of glyphs: %d != %d", nb, seg.NumGlyphs)
	}
	return nil
}

func TestShapeSegment(t *testing.T) {
	for _, input := range fonttestInput {
		face := loadGraphite(t, "testdata/"+input.fontfile)

		expected, err := ioutil.ReadFile("testdata/shape_refs/" + input.name + ".log")
		if err != nil {
			t.Fatal(err)
		}

		out := "Text codes\n"
		for _, r := range input.text {
			out += fmt.Sprintf("%4x\t", r)
		}
		out += "\n"

		feats, outFeats, err := parseFeatures(face, input.features)
		if err != nil {
			t.Fatal(err)
		}
		out += string(outFeats)

		const (
			pointSize = 12
			dpi       = 72
		)
		font := newFontOptions(pointSize*dpi/72, face)
		seg := face.Shape(font, input.text, 0, feats, int8(boolToInt(input.rtl)))

		if err = checkSegmentNumGlyphs(seg); err != nil {
			t.Fatalf("test %s: %s", input.name, err)
		}

		opts := testOptions{input: input.text}
		segString, err := opts.dumpSegment(seg)
		if err != nil {
			t.Fatal(err)
		}
		out += string(segString)

		if out != string(expected) {
			t.Fatalf("for test %s, expected\n%s\n got \n%s\n", input.name, expected, out)
		}
	}
}
