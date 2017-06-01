package generator

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	bsonTagPattern        = "@bson_tag: (.*)"
	bsonCompatiblePattern = "@bson_compatible"
	bsonUpsertablePattern = "@bson_upsertable"
	goInjectPattern       = `(?s)@go_inject\s(.+)`
)

var bsonTagRegex, bsonCompatibleRegex, bsonUpsertableRegex, goInjectRegex *regexp.Regexp

func init() {
	bsonTagRegex = regexp.MustCompile(bsonTagPattern)
	bsonCompatibleRegex = regexp.MustCompile(bsonCompatiblePattern)
	bsonUpsertableRegex = regexp.MustCompile(bsonUpsertablePattern)
	goInjectRegex = regexp.MustCompile(goInjectPattern)
}

func (g *Generator) IsMessageBsonCompatible(message *Descriptor) bool {
	if loc, ok := g.file.comments[message.path]; ok {
		preMessageComments := strings.TrimSuffix(loc.GetLeadingComments(), "\n")
		return bsonCompatibleRegex.Match([]byte(preMessageComments))
	}

	return false
}

func (g *Generator) IsMessageBsonUpsertable(message *Descriptor) bool {
	if loc, ok := g.file.comments[message.path]; ok {
		preMessageComments := strings.TrimSuffix(loc.GetLeadingComments(), "\n")
		return bsonUpsertableRegex.Match([]byte(preMessageComments))
	}

	return false
}

func (g *Generator) GetBsonTagForField(message *Descriptor, fieldNumber int) string {
	fieldPath := fmt.Sprintf("%s,%d,%d", message.path, messageFieldPath, fieldNumber)
	if loc, ok := g.file.comments[fieldPath]; ok {
		comment := strings.TrimSuffix(loc.GetTrailingComments(), "\n")
		matchedGroups := bsonTagRegex.FindStringSubmatch(comment)
		if matchedGroups == nil {
			return ""
		}

		return strings.TrimSpace(matchedGroups[1])
	}

	return ""
}

func (g *Generator) GetGoInjectForMessage(message *Descriptor) string {
	if loc, ok := g.file.comments[message.path]; ok {
		allLeadingComments := loc.GetLeadingDetachedComments()
		allLeadingComments = append(allLeadingComments, loc.GetLeadingComments())
		fmt.Fprintf(os.Stderr, "ALL LEDING: %q\n", allLeadingComments)

		for _, leadingComment := range allLeadingComments {
			matchedGroups := goInjectRegex.FindStringSubmatch(leadingComment)
			if matchedGroups != nil {
				return matchedGroups[1]
			}
		}
	}

	return ""
}
