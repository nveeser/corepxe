package coreos

import (
	"github.com/nveeser/corepxe/server"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TODO test bad requests
// TODO test query params / defaults

type mirrorFunc func(w http.ResponseWriter, r *http.Request, asset server.ImageAsset)

func (m mirrorFunc) ServeAsset(w http.ResponseWriter, r *http.Request, asset server.ImageAsset) {
	m(w, r, asset)
}

func TestImagePathRewrite(t *testing.T) {
	var gotPath string
	mirror := mirrorFunc(func(w http.ResponseWriter, r *http.Request, asset server.ImageAsset) {
		gotPath = asset.RelativePath()
		w.WriteHeader(http.StatusOK)
	})

	h := http.NewServeMux()
	h.Handle("GET /images/coreos/{filetype}", &CoreosImageHandler{mirror})

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "kernel",
			input: "/images/coreos/kernel?version=40.1234.001",
			want:  "coreos/fedora-coreos-40.1234.001-live-kernel-x86_64",
		},
		{
			name:  "rootfs",
			input: "/images/coreos/rootfs?version=40.1234.001",
			want:  "coreos/fedora-coreos-40.1234.001-live-rootfs.x86_64.img",
		},
		{
			name:  "initrd",
			input: "/images/coreos/initrd?version=40.1234.001",
			want:  "coreos/fedora-coreos-40.1234.001-live-initramfs.x86_64.img",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", tc.input, nil)

			h.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Errorf("ServeHTTP() got status %d wanted %d", w.Code, http.StatusOK)
				t.Logf("Body:\n %s\n", w.Body.String())
			}
			if gotPath != tc.want {
				t.Errorf("Path \n\t   got %s \n\twanted %s", gotPath, tc.want)
			}
		})
	}
}
