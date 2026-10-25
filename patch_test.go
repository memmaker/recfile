package recfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A save must keep the file as the author wrote it: comments, blank lines and untouched fields stay put.
func TestPatchKeepsTheRestOfTheFile(t *testing.T) {
	src := "%rec: default\n# docs: every field\n#   icon  the tile\n\ninternal_name: a\n# a note on a\nicon: 1\nname: A\n\ninternal_name: b\nname: B\nsound: x.ogg\nsound: y.ogg\n# trailing\n"
	path := filepath.Join(t.TempDir(), "t.rec")
	os.WriteFile(path, []byte(src), 0o644)
	same := func(id string) ([]string, bool) { return map[string][]string{"a": {"1"}}[id], id == "a" }
	if err := PatchFieldInFile(path, "internal_name", "icon", same); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != src {
		t.Fatalf("an unchanged save rewrote the file:\n%s", got)
	}
	PatchFieldInFile(path, "internal_name", "icon", func(id string) ([]string, bool) { return []string{"7"}, id == "a" })
	PatchFieldInFile(path, "internal_name", "sound", func(id string) ([]string, bool) { return []string{"z.ogg"}, id == "b" })
	want := strings.Replace(strings.Replace(src, "icon: 1", "icon: 7", 1), "sound: x.ogg\nsound: y.ogg", "sound: z.ogg", 1)
	if got, _ := os.ReadFile(path); string(got) != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

// The real files: re-saving them unchanged must not change a byte.
func TestPatchTypedFieldInFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "typed.rec")
	os.WriteFile(path, []byte("%rec: monster\n%doc: Monsters\n\nid: door\nicon: 1\n\n%rec: world\n\nid: door\nicon: 2\nanim: x\n"), 0o644)
	PatchTypedFieldInFile(path, "id", "icon", func(recType, id string) ([]string, bool) {
		return []string{"9"}, recType == "world"
	})
	PatchTypedFieldInFile(path, "id", "anim", func(recType, id string) ([]string, bool) { return nil, true })
	got, _ := os.ReadFile(path)
	want := "%rec: monster\n%doc: Monsters\n\nid: door\nicon: 1\n\n%rec: world\n\nid: door\nicon: 9\n"
	if string(got) != want {
		t.Fatalf("got %q", got)
	}
}

// VoxelRogue's weapons.rec and gear.rec hold every field of a weapon or an armour (dice, attacks repeated per
// line, nine-number hand transforms, comments documenting them); the remapper knows only id, name and icon. A
// save must keep all of it: unchanged, the file stays byte for byte; with new icons, only the icon lines change.
func TestPatchKeepsVoxelRogueFields(t *testing.T) {
	for _, f := range []string{"weapons.rec", "gear.rec", "potions.rec"} {
		orig, err := os.ReadFile(os.ExpandEnv("$HOME/Projects/VoxelRogue/assets/tiles/" + f))
		if err != nil {
			t.Skip(err)
		}
		cp := filepath.Join(t.TempDir(), f)
		os.WriteFile(cp, orig, 0o644)
		sections := ReadMulti(strings.NewReader(string(orig)))
		icons := map[string]string{}
		for typ, recs := range sections {
			for _, r := range recs {
				m := r.ToMap("|")
				if m["id"] != "" && typ != "Tileset" {
					icons[typ+"/"+m["id"]] = m["icon"]
				}
			}
		}
		if len(sections["weapon"])+len(sections["gear"])+len(sections["potion"]) < 8 || len(icons) < 8 {
			t.Fatalf("%s: read %d weapons, %d gear, %d icons", f, len(sections["weapon"]), len(sections["gear"]), len(icons))
		}
		same := func(typ, id string) ([]string, bool) { v, ok := icons[typ+"/"+id]; return []string{v}, ok }
		if err := PatchTypedFieldInFile(cp, "id", "icon", same); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(cp); string(got) != string(orig) {
			t.Fatalf("%s changed on an unchanged save", f)
		}
		PatchTypedFieldInFile(cp, "id", "icon", func(typ, id string) ([]string, bool) { _, ok := icons[typ+"/"+id]; return []string{"77"}, ok })
		got, _ := os.ReadFile(cp)
		var want []string
		for _, l := range strings.Split(string(orig), "\n") {
			if name, _, ok := fieldLine(l); ok && name == "icon" {
				l = "icon: 77"
			}
			want = append(want, l)
		}
		if string(got) != strings.Join(want, "\n") {
			t.Fatalf("%s: a save with new icons changed more than the icon lines", f)
		}
	}
}
