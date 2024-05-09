package sprout

import (
	"bytes"
	"crypto/x509"
	"encoding/base32"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
	bcrypt_lib "golang.org/x/crypto/bcrypt"
)

var urlTests = map[string]map[string]interface{}{
	"proto://auth@host:80/path?query#fragment": {
		"fragment": "fragment",
		"host":     "host:80",
		"hostname": "host",
		"opaque":   "",
		"path":     "/path",
		"query":    "query",
		"scheme":   "proto",
		"userinfo": "auth",
	},
	"proto://host:80/path": {
		"fragment": "",
		"host":     "host:80",
		"hostname": "host",
		"opaque":   "",
		"path":     "/path",
		"query":    "",
		"scheme":   "proto",
		"userinfo": "",
	},
	"something": {
		"fragment": "",
		"host":     "",
		"hostname": "",
		"opaque":   "",
		"path":     "something",
		"query":    "",
		"scheme":   "",
		"userinfo": "",
	},
	"proto://user:passwor%20d@host:80/path": {
		"fragment": "",
		"host":     "host:80",
		"hostname": "host",
		"opaque":   "",
		"path":     "/path",
		"query":    "",
		"scheme":   "proto",
		"userinfo": "user:passwor%20d",
	},
	"proto://host:80/pa%20th?key=val%20ue": {
		"fragment": "",
		"host":     "host:80",
		"hostname": "host",
		"opaque":   "",
		"path":     "/pa th",
		"query":    "key=val%20ue",
		"scheme":   "proto",
		"userinfo": "",
	},
}

func TestUrlParse(t *testing.T) {
	// testing that function is exported and working properly
	assert.NoError(t, runt(
		`{{ index ( urlParse "proto://auth@host:80/path?query#fragment" ) "host" }}`,
		"host:80"))

	// testing scenarios
	fh := NewFunctionHandler()
	for url, expected := range urlTests {
		assert.EqualValues(t, expected, fh.UrlParse(url))
	}
}

func TestUrlJoin(t *testing.T) {
	tests := map[string]string{
		`{{ urlJoin (dict "fragment" "fragment" "host" "host:80" "path" "/path" "query" "query" "scheme" "proto") }}`:       "proto://host:80/path?query#fragment",
		`{{ urlJoin (dict "fragment" "fragment" "host" "host:80" "path" "/path" "scheme" "proto" "userinfo" "ASDJKJSD") }}`: "proto://ASDJKJSD@host:80/path#fragment",
	}
	for tpl, expected := range tests {
		assert.NoError(t, runt(tpl, expected))
	}

	fh := NewFunctionHandler()
	for expected, urlMap := range urlTests {
		assert.EqualValues(t, expected, fh.UrlJoin(urlMap))
	}

}

func TestToString(t *testing.T) {
	tpl := `{{ toString 1 | kindOf }}`
	assert.NoError(t, runt(tpl, "string"))
}

func TestToStrings(t *testing.T) {
	tpl := `{{ $s := list 1 2 3 | toStrings }}{{ index $s 1 | kindOf }}`
	assert.NoError(t, runt(tpl, "string"))
	tpl = `{{ list 1 .value 2 | toStrings }}`
	values := map[string]interface{}{"value": nil}
	if err := runtv(tpl, `[1 2]`, values); err != nil {
		t.Error(err)
	}
}

func TestBase64EncodeDecode(t *testing.T) {
	magicWord := "coffee"
	expect := base64.StdEncoding.EncodeToString([]byte(magicWord))

	if expect == magicWord {
		t.Fatal("Encoder doesn't work.")
	}

	tpl := `{{b64enc "coffee"}}`
	if err := runt(tpl, expect); err != nil {
		t.Error(err)
	}
	tpl = fmt.Sprintf("{{b64dec %q}}", expect)
	if err := runt(tpl, magicWord); err != nil {
		t.Error(err)
	}
}
func TestBase32EncodeDecode(t *testing.T) {
	magicWord := "coffee"
	expect := base32.StdEncoding.EncodeToString([]byte(magicWord))

	if expect == magicWord {
		t.Fatal("Encoder doesn't work.")
	}

	tpl := `{{b32enc "coffee"}}`
	if err := runt(tpl, expect); err != nil {
		t.Error(err)
	}
	tpl = fmt.Sprintf("{{b32dec %q}}", expect)
	if err := runt(tpl, magicWord); err != nil {
		t.Error(err)
	}
}

func TestSemverCompare(t *testing.T) {
	tests := map[string]string{
		`{{ semverCompare "1.2.3" "1.2.3" }}`:  `true`,
		`{{ semverCompare "^1.2.0" "1.2.3" }}`: `true`,
		`{{ semverCompare "^1.2.0" "2.2.3" }}`: `false`,
	}
	for tpl, expect := range tests {
		assert.NoError(t, runt(tpl, expect))
	}
}

func TestSemver(t *testing.T) {
	tests := map[string]string{
		`{{ $s := semver "1.2.3-beta.1+c0ff33" }}{{ $s.Prerelease }}`: "beta.1",
		`{{ $s := semver "1.2.3-beta.1+c0ff33" }}{{ $s.Major}}`:       "1",
		`{{ semver "1.2.3" | (semver "1.2.3").Compare }}`:             `0`,
		`{{ semver "1.2.3" | (semver "1.3.3").Compare }}`:             `1`,
		`{{ semver "1.4.3" | (semver "1.2.3").Compare }}`:             `-1`,
	}
	for tpl, expect := range tests {
		assert.NoError(t, runt(tpl, expect))
	}
}

func TestBiggest(t *testing.T) {
	tpl := `{{ biggest 1 2 3 345 5 6 7}}`
	if err := runt(tpl, `345`); err != nil {
		t.Error(err)
	}

	tpl = `{{ max 345}}`
	if err := runt(tpl, `345`); err != nil {
		t.Error(err)
	}
}

func TestToFloat64(t *testing.T) {
	fh := NewFunctionHandler()
	target := float64(102)
	if target != fh.ToFloat64(int8(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToFloat64(int(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToFloat64(int32(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToFloat64(int16(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToFloat64(int64(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToFloat64("102") {
		t.Errorf("Expected 102")
	}
	if 0 != fh.ToFloat64("frankie") {
		t.Errorf("Expected 0")
	}
	if target != fh.ToFloat64(uint16(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToFloat64(uint64(102)) {
		t.Errorf("Expected 102")
	}
	if 102.1234 != fh.ToFloat64(float64(102.1234)) {
		t.Errorf("Expected 102.1234")
	}
	if 1 != fh.ToFloat64(true) {
		t.Errorf("Expected 102")
	}
}
func TestToInt64(t *testing.T) {
	fh := NewFunctionHandler()
	target := int64(102)
	if target != fh.ToInt64(int8(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt64(int(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt64(int32(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt64(int16(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt64(int64(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt64("102") {
		t.Errorf("Expected 102")
	}
	if 0 != fh.ToInt64("frankie") {
		t.Errorf("Expected 0")
	}
	if target != fh.ToInt64(uint16(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt64(uint64(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt64(float64(102.1234)) {
		t.Errorf("Expected 102")
	}
	if 1 != fh.ToInt64(true) {
		t.Errorf("Expected 102")
	}
}

func TestToInt(t *testing.T) {
	fh := NewFunctionHandler()
	target := int(102)
	if target != fh.ToInt(int8(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt(int(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt(int32(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt(int16(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt(int64(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt("102") {
		t.Errorf("Expected 102")
	}
	if 0 != fh.ToInt("frankie") {
		t.Errorf("Expected 0")
	}
	if target != fh.ToInt(uint16(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt(uint64(102)) {
		t.Errorf("Expected 102")
	}
	if target != fh.ToInt(float64(102.1234)) {
		t.Errorf("Expected 102")
	}
	if 1 != fh.ToInt(true) {
		t.Errorf("Expected 102")
	}
}

func TestToDecimal(t *testing.T) {
	tests := map[interface{}]int64{
		"777": 511,
		777:   511,
		770:   504,
		755:   493,
	}

	for input, expectedResult := range tests {
		result := NewFunctionHandler().ToDecimal(input)
		if result != expectedResult {
			t.Errorf("Expected %v but got %v", expectedResult, result)
		}
	}
}

func TestRandomInt(t *testing.T) {
	var tests = []struct {
		min int
		max int
	}{
		{10, 11},
		{10, 13},
		{0, 1},
		{5, 50},
	}
	for _, v := range tests {
		x, _ := runRaw(fmt.Sprintf(`{{ randInt %d %d }}`, v.min, v.max), nil)
		r, err := strconv.Atoi(x)
		assert.NoError(t, err)
		assert.True(t, func(min, max, r int) bool {
			return r >= v.min && r < v.max
		}(v.min, v.max, r))
	}
}

func TestGetHostByName(t *testing.T) {
	tpl := `{{"www.google.com" | getHostByName}}`

	resolvedIP, _ := runRaw(tpl, nil)

	ip := net.ParseIP(resolvedIP)
	assert.NotNil(t, ip)
	assert.NotEmpty(t, ip)
}

func TestTuple(t *testing.T) {
	tpl := `{{$t := tuple 1 "a" "foo"}}{{index $t 2}}{{index $t 0 }}{{index $t 1}}`
	if err := runt(tpl, "foo1a"); err != nil {
		t.Error(err)
	}
}

func TestIssue188(t *testing.T) {
	tests := map[string]string{

		// This first test shows two merges and the merge is NOT A DEEP COPY MERGE.
		// The first merge puts $one on to $target. When the second merge of $two
		// on to $target the nested dict brought over from $one is changed on
		// $one as well as $target.
		`{{- $target := dict -}}
			{{- $one := dict "foo" (dict "bar" "baz") "qux" true -}}
			{{- $two := dict "foo" (dict "bar" "baz2") "qux" false -}}
			{{- mergeOverwrite $target $one | toString | trunc 0 }}{{ $__ := mergeOverwrite $target $two }}{{ $one }}`: "map[foo:map[bar:baz2] qux:true]",

		// This test uses deepCopy on $one to create a deep copy and then merge
		// that. In this case the merge of $two on to $target does not affect
		// $one because a deep copy was used for that merge.
		`{{- $target := dict -}}
			{{- $one := dict "foo" (dict "bar" "baz") "qux" true -}}
			{{- $two := dict "foo" (dict "bar" "baz2") "qux" false -}}
			{{- deepCopy $one | mergeOverwrite $target | toString | trunc 0 }}{{ $__ := mergeOverwrite $target $two }}{{ $one }}`: "map[foo:map[bar:baz] qux:true]",
	}

	for tpl, expect := range tests {
		if err := runt(tpl, expect); err != nil {
			t.Error(err)
		}
	}
}

// runt runs a template and checks that the output exactly matches the expected string.
func runt(tpl, expect string) error {
	return runtv(tpl, expect, map[string]string{})
}

// runtv takes a template, and expected return, and values for substitution.
//
// It runs the template and verifies that the output is an exact match.
func runtv(tpl, expect string, vars interface{}) error {
	t := template.Must(template.New("test").Funcs(FuncMap()).Parse(tpl))
	var b bytes.Buffer
	err := t.Execute(&b, vars)
	if err != nil {
		return err
	}
	if expect != b.String() {
		return fmt.Errorf("Expected '%v', got '%v'", expect, b.String())
	}
	return nil
}

// runRaw runs a template with the given variables and returns the result.
func runRaw(tpl string, vars interface{}) (string, error) {
	t := template.Must(template.New("test").Funcs(FuncMap()).Parse(tpl))
	var b bytes.Buffer
	err := t.Execute(&b, vars)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func Example() {
	// Set up variables and template.
	vars := map[string]interface{}{"Name": "  John Jacob Jingleheimer Schmidt "}
	tpl := `Hello {{.Name | trim | lower}}`

	// Get the sprout function map.
	t := template.Must(template.New("test").Funcs(FuncMap()).Parse(tpl))

	err := t.Execute(os.Stdout, vars)
	if err != nil {
		fmt.Printf("Error during template execution: %s", err)
		return
	}
	// Output:
	// Hello john jacob jingleheimer schmidt
}

func TestToDate(t *testing.T) {
	tpl := `{{toDate "2006-01-02" "2017-12-31" | date "02/01/2006"}}`
	if err := runt(tpl, "31/12/2017"); err != nil {
		t.Error(err)
	}
}

const (
	beginCertificate = "-----BEGIN CERTIFICATE-----"
	endCertificate   = "-----END CERTIFICATE-----"
)

var (
	// fastCertKeyAlgos is the list of private key algorithms that are supported for certificate use, and
	// are fast to generate.
	fastCertKeyAlgos = []string{
		"ecdsa",
		"ed25519",
	}
)

func TestSha256Sum(t *testing.T) {
	tpl := `{{"abc" | sha256sum}}`
	if err := runt(tpl, "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"); err != nil {
		t.Error(err)
	}
}
func TestSha1Sum(t *testing.T) {
	tpl := `{{"abc" | sha1sum}}`
	if err := runt(tpl, "a9993e364706816aba3e25717850c26c9cd0d89d"); err != nil {
		t.Error(err)
	}
}

func TestAdler32Sum(t *testing.T) {
	tpl := `{{"abc" | adler32sum}}`
	if err := runt(tpl, "38600999"); err != nil {
		t.Error(err)
	}
}

func TestBcrypt(t *testing.T) {
	out, err := runRaw(`{{"abc" | bcrypt}}`, nil)
	if err != nil {
		t.Error(err)
	}
	if bcrypt_lib.CompareHashAndPassword([]byte(out), []byte("abc")) != nil {
		t.Error("Generated hash is not the equivalent for password:", "abc")
	}
}

type HtpasswdCred struct {
	Username string
	Password string
	Valid    bool
}

func TestHtpasswd(t *testing.T) {
	expectations := []HtpasswdCred{
		{Username: "myUser", Password: "myPassword", Valid: true},
		{Username: "special'o79Cv_*qFe,)<user", Password: "special<j7+3p#6-.Jx2U:m8G;kGypassword", Valid: true},
		{Username: "wrongus:er", Password: "doesn'tmatter", Valid: false}, // ':' isn't allowed in the username - https://tools.ietf.org/html/rfc2617#page-6
	}

	for _, credential := range expectations {
		out, err := runRaw(`{{htpasswd .Username .Password}}`, credential)
		if err != nil {
			t.Error(err)
		}
		result := strings.Split(out, ":")
		if 0 != strings.Compare(credential.Username, result[0]) && credential.Valid {
			t.Error("Generated username did not match for:", credential.Username)
		}
		if bcrypt_lib.CompareHashAndPassword([]byte(result[1]), []byte(credential.Password)) != nil && credential.Valid {
			t.Error("Generated hash is not the equivalent for password:", credential.Password)
		}
	}
}

func TestDerivePassword(t *testing.T) {
	expectations := map[string]string{
		`{{derivePassword 1 "long" "password" "user" "example.com"}}`:    "ZedaFaxcZaso9*",
		`{{derivePassword 2 "long" "password" "user" "example.com"}}`:    "Fovi2@JifpTupx",
		`{{derivePassword 1 "maximum" "password" "user" "example.com"}}`: "pf4zS1LjCg&LjhsZ7T2~",
		`{{derivePassword 1 "medium" "password" "user" "example.com"}}`:  "ZedJuz8$",
		`{{derivePassword 1 "basic" "password" "user" "example.com"}}`:   "pIS54PLs",
		`{{derivePassword 1 "short" "password" "user" "example.com"}}`:   "Zed5",
		`{{derivePassword 1 "pin" "password" "user" "example.com"}}`:     "6685",
	}

	for tpl, result := range expectations {
		out, err := runRaw(tpl, nil)
		if err != nil {
			t.Error(err)
		}
		if 0 != strings.Compare(out, result) {
			t.Error("Generated password does not match for", tpl)
		}
	}
}

// NOTE(bacongobbler): this test is really _slow_ because of how long it takes to compute
// and generate a new crypto key.
func TestGenPrivateKey(t *testing.T) {
	// test that calling by default generates an RSA private key
	tpl := `{{genPrivateKey ""}}`
	out, err := runRaw(tpl, nil)
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(out, "RSA PRIVATE KEY") {
		t.Error("Expected RSA PRIVATE KEY")
	}
	// test all acceptable arguments
	tpl = `{{genPrivateKey "rsa"}}`
	out, err = runRaw(tpl, nil)
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(out, "RSA PRIVATE KEY") {
		t.Error("Expected RSA PRIVATE KEY")
	}
	tpl = `{{genPrivateKey "dsa"}}`
	out, err = runRaw(tpl, nil)
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(out, "DSA PRIVATE KEY") {
		t.Error("Expected DSA PRIVATE KEY")
	}
	tpl = `{{genPrivateKey "ecdsa"}}`
	out, err = runRaw(tpl, nil)
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(out, "EC PRIVATE KEY") {
		t.Error("Expected EC PRIVATE KEY")
	}
	tpl = `{{genPrivateKey "ed25519"}}`
	out, err = runRaw(tpl, nil)
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(out, "PRIVATE KEY") {
		t.Error("Expected PRIVATE KEY")
	}
	// test bad
	tpl = `{{genPrivateKey "bad"}}`
	out, err = runRaw(tpl, nil)
	if err != nil {
		t.Error(err)
	}
	if out != "Unknown type bad" {
		t.Error("Expected type 'bad' to be an unknown crypto algorithm")
	}
	// ensure that we can base64 encode the string
	tpl = `{{genPrivateKey "rsa" | b64enc}}`
	_, err = runRaw(tpl, nil)
	if err != nil {
		t.Error(err)
	}
}

func TestUUIDGeneration(t *testing.T) {
	tpl := `{{uuidv4}}`
	out, err := runRaw(tpl, nil)
	if err != nil {
		t.Error(err)
	}

	if len(out) != 36 {
		t.Error("Expected UUID of length 36")
	}

	out2, err := runRaw(tpl, nil)
	if err != nil {
		t.Error(err)
	}

	if out == out2 {
		t.Error("Expected subsequent UUID generations to be different")
	}
}

func TestBuildCustomCert(t *testing.T) {
	ca, _ := NewFunctionHandler().GenerateCertificateAuthority("example.com", 365)
	tpl := fmt.Sprintf(
		`{{- $ca := buildCustomCert "%s" "%s"}}
{{- $ca.Cert }}`,
		base64.StdEncoding.EncodeToString([]byte(ca.Cert)),
		base64.StdEncoding.EncodeToString([]byte(ca.Key)),
	)
	out, err := runRaw(tpl, nil)
	if err != nil {
		t.Error(err)
	}

	tpl2 := fmt.Sprintf(
		`{{- $ca := buildCustomCert "%s" "%s"}}
{{- $ca.Cert }}`,
		base64.StdEncoding.EncodeToString([]byte("fail")),
		base64.StdEncoding.EncodeToString([]byte(ca.Key)),
	)
	out2, _ := runRaw(tpl2, nil)

	assert.Equal(t, out, ca.Cert)
	assert.NotEqual(t, out2, ca.Cert)
}

func TestGenCA(t *testing.T) {
	testGenCA(t, nil)
}

func TestGenCAWithKey(t *testing.T) {
	for _, keyAlgo := range fastCertKeyAlgos {
		t.Run(keyAlgo, func(t *testing.T) {
			testGenCA(t, &keyAlgo)
		})
	}
}

func testGenCA(t *testing.T, keyAlgo *string) {
	const cn = "foo-ca"

	var genCAExpr string
	if keyAlgo == nil {
		genCAExpr = "genCA"
	} else {
		genCAExpr = fmt.Sprintf(`genPrivateKey "%s" | genCAWithKey`, *keyAlgo)
	}

	tpl := fmt.Sprintf(
		`{{- $ca := %s "%s" 365 }}
{{ $ca.Cert }}
`,
		genCAExpr,
		cn,
	)
	out, err := runRaw(tpl, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Contains(t, out, beginCertificate)
	assert.Contains(t, out, endCertificate)

	decodedCert, _ := pem.Decode([]byte(out))
	assert.Nil(t, err)
	cert, err := x509.ParseCertificate(decodedCert.Bytes)
	assert.Nil(t, err)

	assert.Equal(t, cn, cert.Subject.CommonName)
	assert.True(t, cert.IsCA)
}

func TestGenSelfSignedCert(t *testing.T) {
	testGenSelfSignedCert(t, nil)
}

func TestGenSelfSignedCertWithKey(t *testing.T) {
	for _, keyAlgo := range fastCertKeyAlgos {
		t.Run(keyAlgo, func(t *testing.T) {
			testGenSelfSignedCert(t, &keyAlgo)
		})
	}
}

func testGenSelfSignedCert(t *testing.T, keyAlgo *string) {
	const (
		cn   = "foo.com"
		ip1  = "10.0.0.1"
		ip2  = "10.0.0.2"
		dns1 = "bar.com"
		dns2 = "bat.com"
	)

	var genSelfSignedCertExpr string
	if keyAlgo == nil {
		genSelfSignedCertExpr = "genSelfSignedCert"
	} else {
		genSelfSignedCertExpr = fmt.Sprintf(`genPrivateKey "%s" | genSelfSignedCertWithKey`, *keyAlgo)
	}

	tpl := fmt.Sprintf(
		`{{- $cert := %s "%s" (list "%s" "%s") (list "%s" "%s") 365 }}
{{ $cert.Cert }}`,
		genSelfSignedCertExpr,
		cn,
		ip1,
		ip2,
		dns1,
		dns2,
	)

	out, err := runRaw(tpl, nil)
	if err != nil {
		t.Error(err)
	}
	assert.Contains(t, out, beginCertificate)
	assert.Contains(t, out, endCertificate)

	decodedCert, _ := pem.Decode([]byte(out))
	assert.Nil(t, err)
	cert, err := x509.ParseCertificate(decodedCert.Bytes)
	assert.Nil(t, err)

	assert.Equal(t, cn, cert.Subject.CommonName)
	assert.Equal(t, 1, cert.SerialNumber.Sign())
	assert.Equal(t, 2, len(cert.IPAddresses))
	assert.Equal(t, ip1, cert.IPAddresses[0].String())
	assert.Equal(t, ip2, cert.IPAddresses[1].String())
	assert.Contains(t, cert.DNSNames, dns1)
	assert.Contains(t, cert.DNSNames, dns2)
	assert.False(t, cert.IsCA)
}

func TestGenSignedCert(t *testing.T) {
	testGenSignedCert(t, nil, nil)
}

func TestGenSignedCertWithKey(t *testing.T) {
	for _, caKeyAlgo := range fastCertKeyAlgos {
		for _, certKeyAlgo := range fastCertKeyAlgos {
			t.Run(fmt.Sprintf("%s-%s", caKeyAlgo, certKeyAlgo), func(t *testing.T) {
				testGenSignedCert(t, &caKeyAlgo, &certKeyAlgo)
			})
		}
	}
}

func testGenSignedCert(t *testing.T, caKeyAlgo, certKeyAlgo *string) {
	const (
		cn   = "foo.com"
		ip1  = "10.0.0.1"
		ip2  = "10.0.0.2"
		dns1 = "bar.com"
		dns2 = "bat.com"
	)

	var genCAExpr, genSignedCertExpr string
	if caKeyAlgo == nil {
		genCAExpr = "genCA"
	} else {
		genCAExpr = fmt.Sprintf(`genPrivateKey "%s" | genCAWithKey`, *caKeyAlgo)
	}
	if certKeyAlgo == nil {
		genSignedCertExpr = "genSignedCert"
	} else {
		genSignedCertExpr = fmt.Sprintf(`genPrivateKey "%s" | genSignedCertWithKey`, *certKeyAlgo)
	}

	tpl := fmt.Sprintf(
		`{{- $ca := %s "foo" 365 }}
{{- $cert := %s "%s" (list "%s" "%s") (list "%s" "%s") 365 $ca }}
{{ $cert.Cert }}
`,
		genCAExpr,
		genSignedCertExpr,
		cn,
		ip1,
		ip2,
		dns1,
		dns2,
	)
	out, err := runRaw(tpl, nil)
	if err != nil {
		t.Error(err)
	}

	assert.Contains(t, out, beginCertificate)
	assert.Contains(t, out, endCertificate)

	decodedCert, _ := pem.Decode([]byte(out))
	assert.Nil(t, err)
	cert, err := x509.ParseCertificate(decodedCert.Bytes)
	assert.Nil(t, err)

	assert.Equal(t, cn, cert.Subject.CommonName)
	assert.Equal(t, 1, cert.SerialNumber.Sign())
	assert.Equal(t, 2, len(cert.IPAddresses))
	assert.Equal(t, ip1, cert.IPAddresses[0].String())
	assert.Equal(t, ip2, cert.IPAddresses[1].String())
	assert.Contains(t, cert.DNSNames, dns1)
	assert.Contains(t, cert.DNSNames, dns2)
	assert.False(t, cert.IsCA)
}

func TestEncryptDecryptAES(t *testing.T) {
	tpl := `{{"plaintext" | encryptAES "secretkey" | decryptAES "secretkey"}}`
	if err := runt(tpl, "plaintext"); err != nil {
		t.Error(err)
	}
}
