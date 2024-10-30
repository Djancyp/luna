package internal

func CreateTemplate() Template {
	return Template{
		HTML:     nil,
		CSSLinks: CssLink{
      Href:         "",
      DynamicAttrs: nil,
    },
	}
}
