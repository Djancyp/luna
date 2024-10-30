package utils

import (
	"fmt"
	"strings"
)

func GenerateCssLink(href string, dynamicAttrs map[string]string) string {
	cssLinks := []string{fmt.Sprintf("<link href=\"%s\"", href)}
	for key, value := range dynamicAttrs {
		cssLinks = append(cssLinks, fmt.Sprintf("%s=\"%s\"", key, value))
	}
	cssLinks = append(cssLinks, ">")
	return strings.Join(cssLinks, " ")

}
