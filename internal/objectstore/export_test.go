package objectstore

// ReplaceURLHostForTest exposes replaceURLHost for unit tests.
func ReplaceURLHostForTest(raw, publicEndpoint string) string {
	return replaceURLHost(raw, publicEndpoint)
}
