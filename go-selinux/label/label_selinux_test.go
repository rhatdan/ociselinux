// +build selinux,linux

package label

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/opencontainers/selinux/go-selinux"
)

func TestInit(t *testing.T) {
	if !selinux.GetEnabled() {
		return
	}
	var testNull []string
	_, _, err := InitLabels(testNull)
	if err != nil {
		t.Log("InitLabels Failed")
		t.Fatal(err)
	}
	testDisabled := []string{"disable"}
	roMountLabel := ROMountLabel()
	if roMountLabel == "" {
		t.Errorf("ROMountLabel Failed")
	}
	plabel, _, err := InitLabels(testDisabled)
	if err != nil {
		t.Log("InitLabels Disabled Failed")
		t.Fatal(err)
	}
	if plabel != "" {
		t.Log("InitLabels Disabled Failed")
		t.FailNow()
	}
	testUser := []string{"user:user_u", "role:user_r", "type:user_t", "level:s0:c1,c15"}
	plabel, mlabel, err := InitLabels(testUser)
	if err != nil {
		t.Log("InitLabels User Failed")
		t.Fatal(err)
	}
	if plabel != "user_u:user_r:user_t:s0:c1,c15" || (mlabel != "user_u:object_r:container_file_t:s0:c1,c15" && mlabel != "user_u:object_r:svirt_sandbox_file_t:s0:c1,c15") {
		t.Logf("InitLabels User Match Failed %s, %s", plabel, mlabel)
		t.Log(plabel, mlabel)
		t.Fatal(err)
	}

	testBadData := []string{"user", "role:user_r", "type:user_t", "level:s0:c1,c15"}
	if _, _, err = InitLabels(testBadData); err == nil {
		t.Log("InitLabels Bad Failed")
		t.Fatal(err)
	}
}

func TestDuplicateLabel(t *testing.T) {
	secopt, err := DupSecOpt("system_u:system_r:container_t:s0:c1,c2")
	if err != nil {
		t.Fatal(err)
	}
	for _, opt := range secopt {
		con := strings.SplitN(opt, ":", 2)
		if con[0] == "user" {
			if con[1] != "system_u" {
				t.Errorf("DupSecOpt Failed user incorrect")
			}
			continue
		}
		if con[0] == "role" {
			if con[1] != "system_r" {
				t.Errorf("DupSecOpt Failed role incorrect")
			}
			continue
		}
		if con[0] == "type" {
			if con[1] != "container_t" {
				t.Errorf("DupSecOpt Failed type incorrect")
			}
			continue
		}
		if con[0] == "level" {
			if con[1] != "s0:c1,c2" {
				t.Errorf("DupSecOpt Failed level incorrect")
			}
			continue
		}
		t.Errorf("DupSecOpt Failed invalid field %q", con[0])
	}
	secopt = DisableSecOpt()
	if secopt[0] != "disable" {
		t.Errorf("DisableSecOpt Failed level incorrect %q", secopt[0])
	}
}
func TestRelabel(t *testing.T) {
	if !selinux.GetEnabled() {
		return
	}
	testdir, err := ioutil.TempDir("/tmp", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testdir)
	label := "system_u:object_r:container_file_t:s0:c1,c2"
	if err := Relabel(testdir, "", true); err != nil {
		t.Fatalf("Relabel with no label failed: %v", err)
	}
	if err := Relabel(testdir, label, true); err != nil {
		t.Fatalf("Relabel shared failed: %v", err)
	}
	if err := Relabel(testdir, label, false); err != nil {
		t.Fatalf("Relabel unshared failed: %v", err)
	}
	if err := Relabel("/etc", label, false); err == nil {
		t.Fatalf("Relabel /etc succeeded")
	}
	if err := Relabel("/", label, false); err == nil {
		t.Fatalf("Relabel / succeeded")
	}
	if err := Relabel("/usr", label, false); err == nil {
		t.Fatalf("Relabel /usr succeeded")
	}
	if err := Relabel("/usr/", label, false); err == nil {
		t.Fatalf("Relabel /usr/ succeeded")
	}
	if err := Relabel("/etc/passwd", label, false); err == nil {
		t.Fatalf("Relabel /etc/passwd succeeded")
	}
	if home := os.Getenv("HOME"); home != "" {
		if err := Relabel(home, label, false); err == nil {
			t.Fatalf("Relabel %s succeeded", home)
		}
	}
}
func TestValidate(t *testing.T) {
	if err := Validate("zZ"); err != ErrIncompatibleLabel {
		t.Fatalf("Expected incompatible error, got %v", err)
	}
	if err := Validate("Z"); err != nil {
		t.Fatal(err)
	}
	if err := Validate("z"); err != nil {
		t.Fatal(err)
	}
	if err := Validate(""); err != nil {
		t.Fatal(err)
	}
}

func TestIsShared(t *testing.T) {
	if shared := IsShared("Z"); shared {
		t.Fatalf("Expected label `Z` to not be shared, got %v", shared)
	}
	if shared := IsShared("z"); !shared {
		t.Fatalf("Expected label `z` to be shared, got %v", shared)
	}
	if shared := IsShared("Zz"); !shared {
		t.Fatalf("Expected label `Zz` to be shared, got %v", shared)
	}
}

func TestSELinuxNoLevel(t *testing.T) {
	if !selinux.GetEnabled() {
		return
	}
	tlabel := "system_u:system_r:container_t"
	dup, err := DupSecOpt(tlabel)
	if err != nil {
		t.Fatal(err)
	}

	if len(dup) != 3 {
		t.Errorf("DupSecOpt Failed on non mls label")
	}
	con, err := selinux.NewContext(tlabel)
	if err != nil {
		t.Fatal(err)
	}
	if con.Get() != tlabel {
		t.Errorf("NewContaxt and con.Get() Failed on non mls label")
	}
}

func TestSocketLabel(t *testing.T) {
	if !selinux.GetEnabled() {
		return
	}
	label := "system_u:object_r:container_t:s0:c1,c2"
	if err := selinux.SetSocketLabel(label); err != nil {
		t.Fatal(err)
	}
	nlabel, err := selinux.SocketLabel()
	if err != nil {
		t.Fatal(err)
	}
	if label != nlabel {
		t.Errorf("SocketLabel %s != %s", nlabel, label)
	}
}

func TestKeyLabel(t *testing.T) {
	if !selinux.GetEnabled() {
		return
	}
	label := "system_u:object_r:container_t:s0:c1,c2"
	if err := selinux.SetKeyLabel(label); err != nil {
		t.Fatal(err)
	}
	nlabel, err := selinux.KeyLabel()
	if err != nil {
		t.Fatal(err)
	}
	if label != nlabel {
		t.Errorf("KeyLabel %s != %s", nlabel, label)
	}
}

func TestFileLabel(t *testing.T) {
	if !selinux.GetEnabled() {
		return
	}
	testUser := []string{"filetype:test_file_t", "level:s0:c1,c15"}
	_, mlabel, err := InitLabels(testUser)
	if err != nil {
		t.Log("InitLabels User Failed")
		t.Fatal(err)
	}
	if mlabel != "system_u:object_r:test_file_t:s0:c1,c15" {
		t.Log("InitLabels filetype Failed")
		t.Fatal(err)
	}
}
