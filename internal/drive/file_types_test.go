package drive

import "testing"

func TestClassifyFileTypeByName(t *testing.T) {
	_, t1 := ClassifyFileTypeByName("a.jpg", false)
	if t1 != FileTypeImage {
		t.Fatalf("jpg expected image, got %s", t1)
	}
	_, t2 := ClassifyFileTypeByName("a.MP4", false)
	if t2 != FileTypeVideo {
		t.Fatalf("mp4 expected video, got %s", t2)
	}
	_, t3 := ClassifyFileTypeByName("a.pdf", false)
	if t3 != FileTypeDocument {
		t.Fatalf("pdf expected document, got %s", t3)
	}
	_, t4 := ClassifyFileTypeByName("a.zip", false)
	if t4 != FileTypeArchive {
		t.Fatalf("zip expected archive, got %s", t4)
	}
	_, t5 := ClassifyFileTypeByName("a", false)
	if t5 != FileTypeOther {
		t.Fatalf("no ext expected other, got %s", t5)
	}
	_, t6 := ClassifyFileTypeByName("folder", true)
	if t6 != FileTypeFolder {
		t.Fatalf("dir expected folder, got %s", t6)
	}
}

