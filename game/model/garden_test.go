package model

import (
	"testing"
)

func TestDecodeGardenBlobAcceptsEmptyDatabaseValue(t *testing.T) {
	furniture := make(map[int64]*FurnitureDetailsInfo)
	if err := decodeGardenBlob(&furniture, nil); err != nil {
		t.Fatalf("empty garden blob should be valid: %v", err)
	}
	if furniture == nil {
		t.Fatal("empty garden blob must preserve the initialized map")
	}
}

func TestDecodeGardenBlobRejectsMalformedJSON(t *testing.T) {
	var furniture map[int64]*FurnitureDetailsInfo
	if err := decodeGardenBlob(&furniture, []byte("{")); err == nil {
		t.Fatal("malformed garden JSON must be rejected")
	}
}

func TestFurnitureDetailsInfoHandlesMissingTransforms(t *testing.T) {
	info := (&FurnitureDetailsInfo{FurnitureId: 1, FurnitureItemId: 2}).FurnitureDetailsInfo()
	if info.Pos == nil || info.Rotation == nil {
		t.Fatal("missing furniture transforms must default to zero vectors")
	}
}
