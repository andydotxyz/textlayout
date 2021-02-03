package opentype

import "github.com/benoitkugler/textlayout/language"

// const HB_OT_SHAPE_COMPLEX_MAX_COMBINING_MARKS = 32

type hb_ot_shape_zero_width_marks_type_t uint8

const (
	HB_OT_SHAPE_ZERO_WIDTH_MARKS_NONE hb_ot_shape_zero_width_marks_type_t = iota
	HB_OT_SHAPE_ZERO_WIDTH_MARKS_BY_GDEF_EARLY
	HB_OT_SHAPE_ZERO_WIDTH_MARKS_BY_GDEF_LATE
)

type hb_ot_complex_shaper_t interface {
	marksBehavior() (zero_width_marks hb_ot_shape_zero_width_marks_type_t, fallback_position bool)
	normalizationPreference() hb_ot_shape_normalization_mode_t
	// If not 0, then must match found GPOS script tag for
	// GPOS to be applied. Otherwise, fallback positioning will be used.
	gpos_tag() hb_tag_t

	// collect_features is alled during shape_plan().
	// Shapers should use plan.map to add their features and callbacks.
	collect_features(plan *hb_ot_shape_planner_t)

	// override_features is called during shape_plan().
	// Shapers should use plan.map to override features and add callbacks after
	// common features are added.
	override_features(plan *hb_ot_shape_planner_t)

	// data_create is alled at the end of shape_plan().
	// Whatever shapers return will be accessible through plan.data later.
	// If nullptr is returned, means a plan failure.
	data_create(plan *hb_ot_shape_plan_t) interface{}

	// called during shape(), shapers can use to modify text before shaping starts.
	preprocess_text(plan *hb_ot_shape_plan_t, buffer *Buffer, font *Font)

	// called during shape(), shapers can use to modify glyphs after shaping ends.
	postprocess_glyphs(plan *hb_ot_shape_plan_t, buffer *Buffer, font *Font)

	// called during shape()'s normalization: may use decompose_unicode as fallback
	decompose(c *hb_ot_shape_normalize_context_t, ab rune) (a, b rune, ok bool)

	// called during shape()'s normalization: may use compose_unicode as fallback
	compose(c *hb_ot_shape_normalize_context_t, a, b rune) (ab rune, ok bool)

	// called during shape(), shapers should use map to get feature masks and set on buffer.
	// Shapers may NOT modify characters.
	setup_masks(plan *hb_ot_shape_plan_t, buffer *Buffer, font *Font)

	// called during shape(), shapers can use to modify ordering of combining marks.
	reorder_marks(plan *hb_ot_shape_plan_t, buffer *Buffer, start, end int)
}

/*
 * For lack of a better place, put Zawgyi script hack here.
 * https://github.com/harfbuzz/harfbuzz/issues/1162
 */
var scriptMyanmar_Zawgyi = language.Script(newTag('Q', 'a', 'a', 'g'))

func hb_ot_shape_complex_categorize(planner *hb_ot_shape_planner_t) hb_ot_complex_shaper_t {
	switch planner.props.script {
	case language.Arabic, language.Syriac:
		/* For Arabic script, use the Arabic shaper even if no OT script tag was found.
		 * This is because we do fallback shaping for Arabic script (and not others).
		 * But note that Arabic shaping is applicable only to horizontal layout; for
		 * vertical text, just use the generic shaper instead. */
		if (planner.map_.chosen_script[0] != HB_OT_TAG_DEFAULT_SCRIPT ||
			planner.props.script == language.Arabic) &&
			planner.props.direction.isHorizontal() {
			return &_hb_ot_complex_shaper_arabic
		}
		return complexShapedDefault{}
	case language.Thai, language.Lao:
		return &_hb_ot_complex_shaper_thai
	case language.Hangul:
		return &_hb_ot_complex_shaper_hangul
	case language.Hebrew:
		return &_hb_ot_complex_shaper_hebrew
	case language.Bengali, language.Devanagari, language.Gujarati, language.Gurmukhi, language.Kannada,
		language.Malayalam, language.Oriya, language.Tamil, language.Telugu, language.Sinhala:
		/* If the designer designed the font for the 'DFLT' script,
		 * (or we ended up arbitrarily pick 'latn'), use the default shaper.
		 * Otherwise, use the specific shaper.
		 *
		 * If it's indy3 tag, send to USE. */
		if planner.map_.chosen_script[0] == newTag('D', 'F', 'L', 'T') ||
			planner.map_.chosen_script[0] == newTag('l', 'a', 't', 'n') {
			return complexShapedDefault{}
		} else if (planner.map_.chosen_script[0] & 0x000000FF) == '3' {
			return &_hb_ot_complex_shaper_use
		}
		return &_hb_ot_complex_shaper_indic
	case language.Khmer:
		return &_hb_ot_complex_shaper_khmer
	case language.Myanmar:
		/* If the designer designed the font for the 'DFLT' script,
		 * (or we ended up arbitrarily pick 'latn'), use the default shaper.
		 * Otherwise, use the specific shaper.
		 *
		 * If designer designed for 'mymr' tag, also send to default
		 * shaper.  That's tag used from before Myanmar shaping spec
		 * was developed.  The shaping spec uses 'mym2' tag. */
		if planner.map_.chosen_script[0] == newTag('D', 'F', 'L', 'T') ||
			planner.map_.chosen_script[0] == newTag('l', 'a', 't', 'n') ||
			planner.map_.chosen_script[0] == newTag('m', 'y', 'm', 'r') {
			return complexShapedDefault{}
		}
		return &_hb_ot_complex_shaper_myanmar

	/* https://github.com/harfbuzz/harfbuzz/issues/1162 */
	case scriptMyanmar_Zawgyi:
		return &_hb_ot_complex_shaper_myanmar_zawgyi
	case language.Tibetan, language.Mongolian, language.Buhid, language.Hanunoo, language.Tagalog,
		language.Tagbanwa, language.Limbu, language.Tai_Le, language.Buginese, language.Kharoshthi,
		language.Syloti_Nagri, language.Tifinagh, language.Balinese, language.Nko, language.Phags_Pa,
		language.Cham, language.Kayah_Li, language.Lepcha, language.Rejang, language.Saurashtra,
		language.Sundanese, language.Egyptian_Hieroglyphs, language.Javanese, language.Kaithi,
		language.Meetei_Mayek, language.Tai_Tham, language.Tai_Viet, language.Batak,
		language.Brahmi, language.Mandaic, language.Chakma, language.Miao, language.Sharada,
		language.Takri, language.Duployan, language.Grantha, language.Khojki, language.Khudawadi,
		language.Mahajani, language.Manichaean, language.Modi, language.Pahawh_Hmong,
		language.Psalter_Pahlavi, language.Siddham, language.Tirhuta, language.Ahom, language.Multani,
		language.Adlam, language.Bhaiksuki, language.Marchen, language.Newa, language.Masaram_Gondi,
		language.Soyombo, language.Zanabazar_Square, language.Dogra, language.Gunjala_Gondi,
		language.Hanifi_Rohingya, language.Makasar, language.Medefaidrin, language.Old_Sogdian,
		language.Sogdian, language.Elymaic, language.Nandinagari, language.Nyiakeng_Puachue_Hmong,
		language.Wancho, language.Chorasmian, language.Dives_Akuru:

		/* If the designer designed the font for the 'DFLT' script,
		 * (or we ended up arbitrarily pick 'latn'), use the default shaper.
		 * Otherwise, use the specific shaper.
		 * Note that for some simple scripts, there may not be *any*
		 * GSUB/GPOS needed, so there may be no scripts found! */
		if planner.map_.chosen_script[0] == newTag('D', 'F', 'L', 'T') ||
			planner.map_.chosen_script[0] == newTag('l', 'a', 't', 'n') {
			return complexShapedDefault{}
		}
		return &_hb_ot_complex_shaper_use
	default:
		return complexShapedDefault{}
	}
}

type complexShapedDefault struct {
	/* if true, no mark advance zeroing / fallback positioning.
	 * Dumbest shaper ever, basically. */
	dumb bool
}

func (s complexShapedDefault) marksBehavior() (hb_ot_shape_zero_width_marks_type_t, bool) {
	if s.dumb {
		return HB_OT_SHAPE_ZERO_WIDTH_MARKS_NONE, false
	}
	return HB_OT_SHAPE_ZERO_WIDTH_MARKS_BY_GDEF_LATE, true
}

func (complexShapedDefault) normalizationPreference() hb_ot_shape_normalization_mode_t {
	return HB_OT_SHAPE_NORMALIZATION_MODE_DEFAULT
}

func (complexShapedDefault) gpos_tag() hb_tag_t { return 0 }

func (complexShapedDefault) collect_features(plan *hb_ot_shape_planner_t)  {}
func (complexShapedDefault) override_features(plan *hb_ot_shape_planner_t) {}
func (complexShapedDefault) data_create(plan *hb_ot_shape_plan_t) interface{} {
	return nil
}
func (complexShapedDefault) decompose(_ *hb_ot_shape_normalize_context_t, ab rune) (a, b rune, ok bool) {
	return uni.decompose(ab)
}
func (complexShapedDefault) compose(_ *hb_ot_shape_normalize_context_t, a, b rune) (ab rune, ok bool) {
	return uni.compose(a, b)
}
func (complexShapedDefault) preprocess_text(*hb_ot_shape_plan_t, *Buffer, *Font) {}
func (complexShapedDefault) postprocess_glyphs(*hb_ot_shape_plan_t, *Buffer, *Font) {
}
func (complexShapedDefault) setup_masks(*hb_ot_shape_plan_t, *Buffer, *Font)      {}
func (complexShapedDefault) reorder_marks(*hb_ot_shape_plan_t, *Buffer, int, int) {}