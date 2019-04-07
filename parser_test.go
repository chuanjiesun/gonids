/* Copyright 2016 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gonids

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestParseContent(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{
			name:  "simple content",
			input: "abcd",
			want:  []byte("abcd"),
		},
		{
			name:  "escaped content",
			input: `abcd\;ef`,
			want:  []byte("abcd;ef"),
		},
		{
			name:  "hex content",
			input: "A|42 43|D| 45|",
			want:  []byte("ABCDE"),
		},
	} {
		got, err := parseContent(tt.input)
		if !reflect.DeepEqual(got, tt.want) || (err != nil) != tt.wantErr {
			t.Fatalf("%s: got %v,%v; expected %v,%v", tt.name, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestContentToRegexp(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   *Content
		want    string
		wantErr bool
	}{
		{
			name:  "simple content",
			input: &Content{Pattern: []byte("abcd")},
			want:  `abcd`,
		},
		{
			name:  "escaped content",
			input: &Content{Pattern: []byte("abcd;ef")},
			want:  `abcd;ef`,
		},
		{
			name:  "complex escaped content",
			input: &Content{Pattern: []byte("abcd;:\r\ne\rf")},
			want:  `abcd;:\.\.e\.f`,
		},
	} {
		got := tt.input.ToRegexp()
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("%s: got %v; expected %v", tt.name, got, tt.want)
		}
	}
}

func TestContentFormatPattern(t *testing.T) {
	for _, tt := range []struct {
		name    string
		input   *Content
		want    string
		wantErr bool
	}{
		{
			name:  "simple content",
			input: &Content{Pattern: []byte("abcd")},
			want:  "abcd",
		},
		{
			name:  "escaped content",
			input: &Content{Pattern: []byte("abcd;ef")},
			want:  "abcd|3B|ef",
		},
		{
			name:  "complex escaped content",
			input: &Content{Pattern: []byte("abcd;:\r\ne\rf")},
			want:  "abcd|3B 3A 0D 0A|e|0D|f",
		},
	} {
		got := tt.input.FormatPattern()
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("%s: got %v; expected %v", tt.name, got, tt.want)
		}
	}
}

func TestParseRule(t *testing.T) {
	for _, tt := range []struct {
		name    string
		rule    string
		want    *Rule
		wantErr bool
	}{
		{
			name: "simple content",
			rule: `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1337; msg:"foo"; content:"AA"; rev:2);`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "udp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"any"}},
				SID:         1337,
				Revision:    2,
				Description: "foo",
				Contents:    []*Content{&Content{Pattern: []byte{0x41, 0x41}}},
			},
		},
		{
			name: "bidirectional",
			rule: `alert udp $HOME_NET any <> $EXTERNAL_NET any (sid:1337; msg:"foo"; content:"AA"; rev:2);`,
			want: &Rule{
				Action:        "alert",
				Protocol:      "udp",
				Source:        Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination:   Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"any"}},
				Bidirectional: true,
				SID:           1337,
				Revision:      2,
				Description:   "foo",
				Contents:      []*Content{&Content{Pattern: []byte{0x41, 0x41}}},
			},
		},
		{
			name: "not content",
			rule: `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1337; msg:"foo"; content:!"AA");`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "udp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"any"}},
				SID:         1337,
				Description: "foo",
				Contents:    []*Content{&Content{Pattern: []byte{0x41, 0x41}, Negate: true}},
			},
		},
		{
			name: "multiple contents",
			rule: `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1337; msg:"foo"; content:"AA"; content:"BB");`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "udp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"any"}},
				SID:         1337,
				Description: "foo",
				Contents: []*Content{&Content{Pattern: []byte{0x41, 0x41}},
					&Content{Pattern: []byte{0x42, 0x42}}},
			},
		},
		{
			name: "hex content",
			rule: `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1337; msg:"foo"; content:"A|42 43|D|45|");`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "udp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"any"}},
				SID:         1337,
				Description: "foo",
				Contents:    []*Content{&Content{Pattern: []byte{0x41, 0x42, 0x43, 0x44, 0x45}}},
			},
		},
		{
			name: "tags",
			rule: `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1337; msg:"foo"; content:!"AA"; classtype:foo);`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "udp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"any"}},
				SID:         1337,
				Description: "foo",
				Contents:    []*Content{&Content{Pattern: []byte{0x41, 0x41}, Negate: true}},
				Tags:        map[string]string{"classtype": "foo"},
			},
		},
		{
			name: "references",
			rule: `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1337; msg:"foo"; content:"A"; reference:cve,2014; reference:url,www.suricata-ids.org);`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "udp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"any"}},
				SID:         1337,
				Description: "foo",
				Contents:    []*Content{&Content{Pattern: []byte{0x41}}},
				References:  []*Reference{&Reference{Type: "cve", Value: "2014"}, &Reference{Type: "url", Value: "www.suricata-ids.org"}},
			},
		},
		{
			name: "content options",
			rule: `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1337; msg:"foo"; content:!"AA"; http_header; offset:3);`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "udp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"any"}},
				SID:         1337,
				Description: "foo",
				Contents: []*Content{&Content{
					Pattern: []byte{0x41, 0x41},
					Negate:  true,
					Options: []*ContentOption{&ContentOption{"http_header", 0}, &ContentOption{"offset", 3}},
				}},
			},
		},
		{
			name: "multiple contents and options",
			rule: `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1; msg:"a"; content:"A"; http_header; fast_pattern; content:"B"; http_uri);`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "udp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"any"}},
				SID:         1,
				Description: "a",
				Contents: []*Content{
					&Content{Pattern: []byte{0x41}, Options: []*ContentOption{&ContentOption{"http_header", 0}}, FastPattern: FastPattern{Enabled: true}},
					&Content{Pattern: []byte{0x42}, Options: []*ContentOption{&ContentOption{"http_uri", 0}}},
				},
			},
		},
		{
			name: "multiple contents and multiple options",
			rule: `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1; msg:"a"; content:"A"; http_header; fast_pattern:0,42; nocase; content:"B"; http_uri);`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "udp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"any"}},
				SID:         1,
				Description: "a",
				Contents: []*Content{
					&Content{Pattern: []byte{0x41}, Options: []*ContentOption{&ContentOption{"http_header", 0}, &ContentOption{"nocase", 0}}, FastPattern: FastPattern{Enabled: true, Offset: 0, Length: 42}},
					&Content{Pattern: []byte{0x42}, Options: []*ContentOption{&ContentOption{"http_uri", 0}}},
				},
			},
		},
		{
			name: "multiple contents with file_data",
			rule: `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1; msg:"a"; file_data; content:"A"; http_header; nocase; content:"B"; http_uri);`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "udp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"any"}},
				SID:         1,
				Description: "a",
				Contents: []*Content{
					&Content{DataPosition: 1, Pattern: []byte{0x41}, Options: []*ContentOption{&ContentOption{"http_header", 0}, &ContentOption{"nocase", 0}}},
					&Content{DataPosition: 1, Pattern: []byte{0x42}, Options: []*ContentOption{&ContentOption{"http_uri", 0}}},
				},
			},
		},
		{
			name: "multiple contents with file_data and pkt_data",
			rule: `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1; msg:"a"; file_data; content:"A"; http_header; nocase; content:"B"; http_uri; pkt_data; content:"C"; http_uri;)`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "udp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"any"}},
				SID:         1,
				Description: "a",
				Contents: []*Content{
					&Content{DataPosition: 1, Pattern: []byte{0x41}, Options: []*ContentOption{&ContentOption{"http_header", 0}, &ContentOption{"nocase", 0}}},
					&Content{DataPosition: 1, Pattern: []byte{0x42}, Options: []*ContentOption{&ContentOption{"http_uri", 0}}},
					&Content{DataPosition: 0, Pattern: []byte{0x43}, Options: []*ContentOption{&ContentOption{"http_uri", 0}}},
				},
			},
		},
		{
			name: "Complex VRT rule",
			rule: `alert tcp $HOME_NET any -> $EXTERNAL_NET $HTTP_PORTS (msg:"VRT BLACKLIST URI request for known malicious URI - /tongji.js"; flow:to_server,established; content:"/tongji.js"; fast_pattern:only; http_uri; content:"Host|3A| "; http_header; pcre:"/Host\x3a[^\r\n]*?\.tongji/Hi"; metadata:impact_flag red, policy balanced-ips drop, policy security-ips drop, ruleset community, service http; reference:url,labs.snort.org/docs/17904.html; classtype:trojan-activity; sid:17904; rev:6;)`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "tcp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"$HTTP_PORTS"}},
				SID:         17904,
				Revision:    6,
				Description: "VRT BLACKLIST URI request for known malicious URI - /tongji.js",
				References:  []*Reference{&Reference{Type: "url", Value: "labs.snort.org/docs/17904.html"}},
				Contents: []*Content{
					&Content{Pattern: []byte{0x2f, 0x74, 0x6f, 0x6e, 0x67, 0x6a, 0x69, 0x2e, 0x6a, 0x73}, Options: []*ContentOption{&ContentOption{"http_uri", 0}}, FastPattern: FastPattern{Enabled: true, Only: true}},
					&Content{Pattern: []byte{0x48, 0x6f, 0x73, 0x74, 0x3a, 0x20}, Options: []*ContentOption{&ContentOption{"http_header", 0}}},
				},
				PCREs: []*PCRE{
					&PCRE{
						Pattern: []byte{0x48, 0x6f, 0x73, 0x74, 0x5c, 0x78, 0x33, 0x61, 0x5b, 0x5e, 0x5c, 0x72, 0x5c, 0x6e, 0x5d, 0x2a, 0x3f, 0x5c, 0x2e, 0x74, 0x6f, 0x6e, 0x67, 0x6a, 0x69},
						Options: []byte{0x48, 0x69},
					},
				},
				Tags: map[string]string{"flow": "to_server,established", "classtype": "trojan-activity"},
				Metas: []*Metadata{
					&Metadata{Key: "impact_flag", Value: "red"},
					&Metadata{Key: "policy", Value: "balanced-ips drop"},
					&Metadata{Key: "policy", Value: "security-ips drop"},
					&Metadata{Key: "ruleset", Value: "community"},
					&Metadata{Key: "service", Value: "http"},
				},
			},
		},
		{
			name: "content and PCRE",
			rule: `alert tcp $HOME_NET any -> $EXTERNAL_NET $HTTP_PORTS (msg:"Foo msg"; flow:to_server,established; content:"blah"; http_uri; pcre:"/foo.*bar/Ui"; reference:url,www.google.com; classtype:trojan-activity; sid:12345; rev:1;)`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "tcp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"$HTTP_PORTS"}},
				SID:         12345,
				Revision:    1,
				Description: "Foo msg",
				References:  []*Reference{&Reference{Type: "url", Value: "www.google.com"}},
				Contents: []*Content{
					&Content{Pattern: []byte{0x62, 0x6c, 0x61, 0x68}, Options: []*ContentOption{&ContentOption{"http_uri", 0}}},
				},
				PCREs: []*PCRE{
					&PCRE{
						Pattern: []byte{0x66, 0x6f, 0x6f, 0x2e, 0x2a, 0x62, 0x61, 0x72},
						Options: []byte{0x55, 0x69}},
				},
				Tags: map[string]string{"flow": "to_server,established", "classtype": "trojan-activity"},
			},
		},
		{
			name: "Metadata",
			rule: `alert tcp any any -> any any (msg:"ET SHELLCODE Berlin Shellcode"; flow:established; content:"|31 c9 b1 fc 80 73 0c|"; content:"|43 e2 8b 9f|"; distance:0; reference:url,doc.emergingthreats.net/2009256; classtype:shellcode-detect; sid:2009256; rev:3; metadata:created_at 2010_07_30, updated_at 2010_07_30;)`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "tcp",
				Source:      Network{Nets: []string{"any"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"any"}, Ports: []string{"any"}},
				SID:         2009256,
				Revision:    3,
				Description: "ET SHELLCODE Berlin Shellcode",
				References:  []*Reference{&Reference{Type: "url", Value: "doc.emergingthreats.net/2009256"}},
				Contents: []*Content{
					&Content{Pattern: []byte{0x31, 0xc9, 0xb1, 0xfc, 0x80, 0x73, 0x0c}}, 
					&Content{Pattern: []byte{0x43, 0xe2, 0x8b, 0x9f},Options: []*ContentOption{ &ContentOption{"distance", 0}}},
				},
				Tags: map[string]string{"flow": "established", "classtype": "shellcode-detect"},
				Metas: []*Metadata{
					&Metadata{Key: "created_at", Value: "2010_07_30"},
					&Metadata{Key: "updated_at", Value: "2010_07_30"},
				},
			},
		},
		{
			name: "Multi Metadata",
			rule: `alert http $EXTERNAL_NET any -> $HOME_NET any (msg:"ET CURRENT_EVENTS Chase Account Phish Landing Oct 22"; flow:established,from_server; file_data; content:"<title>Sign in</title>"; content:"name=chalbhai"; fast_pattern; nocase; distance:0; content:"required title=|22|Please Enter Right Value|22|"; nocase; distance:0; content:"required title=|22|Please Enter Right Value|22|"; nocase; distance:0; metadata: former_category CURRENT_EVENTS; classtype:trojan-activity; sid:2025692; rev:2; metadata:created_at 2015_10_22, updated_at 2018_07_12;)`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "http",
				Source:      Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				SID:         2025692,
				Revision:    2,
				Description: "ET CURRENT_EVENTS Chase Account Phish Landing Oct 22",
				Contents: []*Content{
					&Content{Pattern: []byte{0x3c, 0x74, 0x69, 0x74, 0x6c, 0x65, 0x3e, 0x53, 0x69, 0x67, 0x6e, 0x20, 0x69, 0x6e, 0x3c, 0x2f, 0x74, 0x69, 0x74, 0x6c, 0x65, 0x3e}, DataPosition:1, FastPattern: FastPattern{Enabled:false, Length:0, Offset:0}, Negate:false}, 
					&Content{Pattern: []byte{0x6e, 0x61, 0x6d, 0x65, 0x3d, 0x63, 0x68, 0x61, 0x6c, 0x62, 0x68, 0x61, 0x69}, DataPosition:1, Options: []*ContentOption{ &ContentOption{"nocase", 0}, &ContentOption{"distance", 0} }, FastPattern: FastPattern{Enabled:true, Length:0, Offset:0}, Negate:false},
					&Content{Pattern: []byte{0x72, 0x65, 0x71, 0x75, 0x69, 0x72, 0x65, 0x64, 0x20, 0x74, 0x69, 0x74, 0x6c, 0x65, 0x3d, 0x22, 0x50, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x20, 0x45, 0x6e, 0x74, 0x65, 0x72, 0x20, 0x52, 0x69, 0x67, 0x68, 0x74, 0x20, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x22}, DataPosition:1, Options: []*ContentOption{ &ContentOption{"nocase", 0},&ContentOption{"distance", 0} }, FastPattern: FastPattern{Enabled:false, Length:0, Offset:0}, Negate:false},
					&Content{Pattern: []byte{0x72, 0x65, 0x71, 0x75, 0x69, 0x72, 0x65, 0x64, 0x20, 0x74, 0x69, 0x74, 0x6c, 0x65, 0x3d, 0x22, 0x50, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x20, 0x45, 0x6e, 0x74, 0x65, 0x72, 0x20, 0x52, 0x69, 0x67, 0x68, 0x74, 0x20, 0x56, 0x61, 0x6c, 0x75, 0x65, 0x22}, DataPosition:1, Options: []*ContentOption{ &ContentOption{"nocase", 0},&ContentOption{"distance", 0} }, FastPattern: FastPattern{Enabled:false, Length:0, Offset:0}, Negate:false},
				},
				Tags: map[string]string{"flow": "established,from_server", "classtype": "trojan-activity"},
				Metas: []*Metadata{
					&Metadata{Key: "former_category", Value: "CURRENT_EVENTS"},
					&Metadata{Key: "created_at", Value: "2015_10_22"},
					&Metadata{Key: "updated_at", Value: "2018_07_12"},
				},
			},
		},
		{
			name: "Negated PCRE",
			rule: `alert tcp $HOME_NET any -> $EXTERNAL_NET $HTTP_PORTS (msg:"Negated PCRE"; pcre:!"/foo.*bar/Ui"; sid:12345; rev:1;)`,
			want: &Rule{
				Action:      "alert",
				Protocol:    "tcp",
				Source:      Network{Nets: []string{"$HOME_NET"}, Ports: []string{"any"}},
				Destination: Network{Nets: []string{"$EXTERNAL_NET"}, Ports: []string{"$HTTP_PORTS"}},
				SID:         12345,
				Revision:    1,
				Description: "Negated PCRE",
				PCREs: []*PCRE{
					&PCRE{
						Pattern: []byte{0x66, 0x6f, 0x6f, 0x2e, 0x2a, 0x62, 0x61, 0x72},
						Negate: true,
						Options: []byte{0x55, 0x69}},
				},
			},
		},
		// Errors
		//TODO: Fix lexer with invalid direction. This test causes an infinite loop.
		//{
			//name:    "invalid direction",
			//rule:    `alert udp $HOME_NET any *# $EXTERNAL_NET any (sid:2; msg:"foo"; content:"A");`,
			//wantErr: true,
		//},
		{
			name:    "invalid sid",
			rule:    `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:"a");`,
			wantErr: true,
		},
		{
			name:    "invalid content option",
			rule:    `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1; content:"foo"; offset:"a");`,
			wantErr: true,
		},
		{
			name:    "invalid content value",
			rule:    `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1; content:!; offset:"a");`,
			wantErr: true,
		},
		{
			name:    "invalid msg",
			rule:    `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:2; msg; content:"A");`,
			wantErr: true,
		},
	} {
		got, err := ParseRule(tt.rule)
		if !reflect.DeepEqual(got, tt.want) || (err != nil) != tt.wantErr {
			t.Fatal(spew.Sprintf("%s: got=%+v,%+v; want=%+v,%+v", tt.name, got, err, tt.want, tt.wantErr))
		}
	}
}

func TestRE(t *testing.T) {
	for _, tt := range []struct {
		rule string
		want string
	}{
		{
			rule: `alert udp $HOME_NET any -> $EXTERNAL_NET any (sid:1337; msg:"foo"; content:"|28|foo"; content:".AA"; within:40);`,
			want: `.*\(foo.{0,40}\.AA`,
		},
	} {
		r, err := ParseRule(tt.rule)
		if err != nil {
			t.Fatalf("re: parse rule failed: %v", err)
		}
		if got := r.RE(); got != tt.want {
			t.Fatalf("re: got=%v; want=%v", got, tt.want)
		}
	}
}
