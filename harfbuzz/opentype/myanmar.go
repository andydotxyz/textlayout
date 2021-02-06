package opentype

import (
	"fmt"

	cm "github.com/benoitkugler/textlayout/harfbuzz/common"
)

// ported from harfbuzz/src/hb-ot-shape-complex-myanmar.cc, .hh Copyright © 2011,2012,2013  Google, Inc.  Behdad Esfahbod

/*
 * Myanmar shaper.
 */
type complexShaperMyanmar struct{}

var _ hb_ot_complex_shaper_t = complexShaperMyanmar{}

/*
 * Basic features.
 * These features are applied in order, one at a time, after reordering.
 */
var myanmar_basic_features = [...]hb_tag_t{
	newTag('r', 'p', 'h', 'f'),
	newTag('p', 'r', 'e', 'f'),
	newTag('b', 'l', 'w', 'f'),
	newTag('p', 's', 't', 'f'),
}

/*
* Other features.
* These features are applied all at once, after clearing syllables.
 */
var myanmar_other_features = [...]hb_tag_t{
	newTag('p', 'r', 'e', 's'),
	newTag('a', 'b', 'v', 's'),
	newTag('b', 'l', 'w', 's'),
	newTag('p', 's', 't', 's'),
}

func (complexShaperMyanmar) collectFeatures(plan *hb_ot_shape_planner_t) {
	map_ := &plan.map_

	/* Do this before any lookups have been applied. */
	map_.add_gsub_pause(setupSyllablesMyanmar)

	map_.enable_feature(newTag('l', 'o', 'c', 'l'))
	/* The Indic specs do not require ccmp, but we apply it here since if
	* there is a use of it, it's typically at the beginning. */
	map_.enable_feature(newTag('c', 'c', 'm', 'p'))

	map_.add_gsub_pause(reorderMyanmar)

	for _, feat := range myanmar_basic_features {
		map_.enable_feature_ext(feat, F_MANUAL_ZWJ, 1)
		map_.add_gsub_pause(nil)
	}

	map_.add_gsub_pause(_hb_clear_syllables)

	for _, feat := range myanmar_other_features {
		map_.enable_feature_ext(feat, F_MANUAL_ZWJ, 1)
	}
}

func (complexShaperMyanmar) setupMasks(_ *hb_ot_shape_plan_t, buffer *cm.Buffer, _ *cm.Font) {
	/* We cannot setup masks here.  We save information about characters
	* and setup masks later on in a pause-callback. */

	info := buffer.Info
	for i := range info {
		setMyanmarProperties(&info[i])
	}
}

func foundSyllableMyanmar(syllableType uint8, ts, te int, info []cm.GlyphInfo, syllableSerial *uint8) {
	for i := ts; i < te; i++ {
		info[i].Aux2 = (*syllableSerial << 4) | syllableType
	}
	*syllableSerial++
	if *syllableSerial == 16 {
		*syllableSerial = 1
	}
}

func setupSyllablesMyanmar(_ *hb_ot_shape_plan_t, _ *cm.Font, buffer *cm.Buffer) {
	findSyllablesMyanmar(buffer)
	iter, count := buffer.SyllableIterator()
	for start, end := iter.Next(); start < count; start, end = iter.Next() {
		buffer.UnsafeToBreak(start, end)
	}
}

/* Rules from:
 * https://docs.microsoft.com/en-us/typography/script-development/myanmar */
func initialReorderingConsonantSyllable(buffer *cm.Buffer, start, end int) {
	info := buffer.Info

	base := end
	hasReph := false

	limit := start
	if start+3 <= end &&
		info[start].AuxCategory == OT_Ra &&
		info[start+1].AuxCategory == OT_As &&
		info[start+2].AuxCategory == OT_H {
		limit += 3
		base = start
		hasReph = true
	}

	{
		if !hasReph {
			base = limit
		}

		for i := limit; i < end; i++ {
			if is_consonant(info[i]) {
				base = i
				break
			}
		}
	}

	/* Reorder! */
	i := start
	end = start
	if hasReph {
		end = start + 3
	}
	for ; i < end; i++ {
		info[i].Aux2 = POS_AFTER_MAIN
	}
	for ; i < base; i++ {
		info[i].Aux2 = POS_PRE_C
	}
	if i < end {
		info[i].Aux2 = POS_BASE_C
		i++
	}
	pos := POS_AFTER_MAIN
	/* The following loop may be ugly, but it implements all of
	 * Myanmar reordering! */
	for ; i < end; i++ {
		if info[i].AuxCategory == OT_MR /* Pre-base reordering */ {
			info[i].Aux2 = POS_PRE_C
			continue
		}
		if info[i].Aux2 < POS_BASE_C /* Left matra */ {
			continue
		}
		if info[i].AuxCategory == OT_VS {
			info[i].Aux2 = info[i-1].Aux2
			continue
		}

		if pos == POS_AFTER_MAIN && info[i].AuxCategory == OT_VBlw {
			pos = POS_BELOW_C
			info[i].Aux2 = pos
			continue
		}

		if pos == POS_BELOW_C && info[i].AuxCategory == OT_A {
			info[i].Aux2 = POS_BEFORE_SUB
			continue
		}
		if pos == POS_BELOW_C && info[i].AuxCategory == OT_VBlw {
			info[i].Aux2 = pos
			continue
		}
		if pos == POS_BELOW_C && info[i].AuxCategory != OT_A {
			pos = POS_AFTER_SUB
			info[i].Aux2 = pos
			continue
		}
		info[i].Aux2 = pos
	}

	/* Sit tight, rock 'n roll! */
	buffer.Sort(start, end, func(a, b *cm.GlyphInfo) int { return int(a.Aux2) - int(b.Aux2) })
}

func reorderSyllableMyanmar(buffer *cm.Buffer, start, end int) {
	syllableType := buffer.Info[start].Aux2 & 0x0F
	switch syllableType {
	/* We already inserted dotted-circles, so just call the consonant_syllable. */
	case myanmarBrokenCluster, myanmarConsonantSyllable:
		initialReorderingConsonantSyllable(buffer, start, end)
	}
}

func reorderMyanmar(plan *hb_ot_shape_plan_t, font *cm.Font, buffer *cm.Buffer) {
	if cm.DebugMode {
		fmt.Println("MYANMAR - start reordering myanmar")
	}

	hb_syllabic_insert_dotted_circles(font, buffer, myanmarBrokenCluster, OT_GB, -1)

	iter, count := buffer.SyllableIterator()
	for start, end := iter.Next(); start < count; start, end = iter.Next() {
		reorderSyllableMyanmar(buffer, start, end)
	}

	if cm.DebugMode {
		fmt.Println("MYANMAR - end reordering myanmar")
	}
}

/* Note: This enum is duplicated in the -machine.rl source file.
 * Not sure how to avoid duplication. */
const (
	OT_As = 18   /* Asat */
	OT_D0 = 20   /* Digit zero */
	OT_DB = OT_N /* Dot below */
	OT_GB = OT_PLACEHOLDER
	OT_MH = 21 /* Various consonant medial types */
	OT_MR = 22 /* Various consonant medial types */
	OT_MW = 23 /* Various consonant medial types */
	OT_MY = 24 /* Various consonant medial types */
	OT_PT = 25 /* Pwo and other tones */
	//OT_VAbv = 26
	//OT_VBlw = 27
	//OT_VPre = 28
	//OT_VPst = 29
	OT_VS = 30 /* Variation selectors */
	OT_P  = 31 /* Punctuation */
	OT_D  = 32 /* Digits except zero */
)

func setMyanmarProperties(info *cm.GlyphInfo) {
	u := info.Codepoint
	type_ := hb_indic_get_categories(u)
	cat := type_ & 0xFF
	pos := type_ >> 8

	/* Myanmar
	* https://docs.microsoft.com/en-us/typography/script-development/myanmar#analyze */
	if 0xFE00 <= u && u <= 0xFE0F {
		cat = OT_VS
	}

	switch u {
	case 0x104E:
		cat = OT_C /* The spec says C, IndicSyllableCategory doesn't have. */
	case 0x002D, 0x00A0, 0x00D7, 0x2012, 0x2013, 0x2014, 0x2015, 0x2022,
		0x25CC, 0x25FB, 0x25FC, 0x25FD, 0x25FE:
		cat = OT_GB
	case 0x1004, 0x101B, 0x105A:
		cat = OT_Ra
	case 0x1032, 0x1036:
		cat = OT_A
	case 0x1039:
		cat = OT_H
	case 0x103A:
		cat = OT_As
	case 0x1041, 0x1042, 0x1043, 0x1044, 0x1045, 0x1046, 0x1047, 0x1048,
		0x1049, 0x1090, 0x1091, 0x1092, 0x1093, 0x1094, 0x1095, 0x1096, 0x1097, 0x1098, 0x1099:
		cat = OT_D
	case 0x1040:
		cat = OT_D /* The spec says D0, but Uniscribe doesn't seem to do. */
	case 0x103E, 0x1060:
		cat = OT_MH
	case 0x103C:
		cat = OT_MR
	case 0x103D, 0x1082:
		cat = OT_MW
	case 0x103B, 0x105E, 0x105F:
		cat = OT_MY
	case 0x1063, 0x1064, 0x1069, 0x106A, 0x106B, 0x106C, 0x106D, 0xAA7B:
		cat = OT_PT
	case 0x1038, 0x1087, 0x1088, 0x1089, 0x108A, 0x108B, 0x108C, 0x108D,
		0x108F, 0x109A, 0x109B, 0x109C:
		cat = OT_SM
	case 0x104A, 0x104B:
		cat = OT_P
	case 0xAA74, 0xAA75, 0xAA76:
		/* https://github.com/harfbuzz/harfbuzz/issues/218 */
		cat = OT_C
	}

	if cat == OT_M {
		switch pos {
		case POS_PRE_C:
			cat = OT_VPre
			pos = POS_PRE_M
		case POS_ABOVE_C:
			cat = OT_VAbv
		case POS_BELOW_C:
			cat = OT_VBlw
		case POS_POST_C:
			cat = OT_VPst
		}
	}

	info.AuxCategory = cat
	info.Aux2 = pos
}

func (complexShaperMyanmar) marksBehavior() (hb_ot_shape_zero_width_marks_type_t, bool) {
	return HB_OT_SHAPE_ZERO_WIDTH_MARKS_BY_GDEF_EARLY, false
}

func (complexShaperMyanmar) normalizationPreference() hb_ot_shape_normalization_mode_t {
	return HB_OT_SHAPE_NORMALIZATION_MODE_COMPOSED_DIACRITICS_NO_SHORT_CIRCUIT
}

func (complexShaperMyanmar) compose(_ *hb_ot_shape_normalize_context_t, a, b rune) (rune, bool) {
	return cm.Uni.Compose(a, b)
}
func (complexShaperMyanmar) decompose(c *hb_ot_shape_normalize_context_t, ab rune) (a, b rune, ok bool) {
	return cm.Uni.Decompose(ab)
}
func (complexShaperMyanmar) gposTag() hb_tag_t { return 0 }
func (complexShaperMyanmar) overrideFeatures(plan *hb_ot_shape_planner_t)
func (complexShaperMyanmar) preprocessText(plan *hb_ot_shape_plan_t, buffer *cm.Buffer, font *cm.Font)
func (complexShaperMyanmar) dataCreate(plan *hb_ot_shape_plan_t)
func (complexShaperMyanmar) reorderMarks(plan *hb_ot_shape_plan_t, buffer *cm.Buffer, start, end int)
func (complexShaperMyanmar) postprocessGlyphs(plan *hb_ot_shape_plan_t, buffer *cm.Buffer, font *cm.Font)
