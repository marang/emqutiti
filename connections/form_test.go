package connections

import (
	"os"
	"strings"
	"testing"

	"github.com/marang/emqutiti/ui"
)

func TestNewConnectionFormEnvReadOnly(t *testing.T) {
	os.Setenv("EMQUTITI_ENV_HOST", "envhost")
	os.Setenv("EMQUTITI_ENV_PORT", "1884")
	t.Cleanup(func() {
		os.Unsetenv("EMQUTITI_ENV_HOST")
		os.Unsetenv("EMQUTITI_ENV_PORT")
	})
	cf := NewForm(Profile{Name: "env", FromEnv: true}, 0)
	hostField, ok := cf.Fields[fieldIndex["Host"]].(*ui.TextField)
	if !ok || hostField.Value() != "envhost" {
		t.Fatalf("host not loaded from env: %v", hostField)
	}
	portField := cf.Fields[fieldIndex["Port"]].(*ui.TextField)
	if portField.Value() != "1884" {
		t.Fatalf("port not loaded from env: %s", portField.Value())
	}
	idxName := fieldIndex["Name"]
	idxFromEnv := fieldIndex["FromEnv"]
	for i, fld := range cf.Fields {
		if i == idxName || i == idxFromEnv {
			continue
		}
		switch f := fld.(type) {
		case *ui.TextField:
			if !f.ReadOnly() {
				t.Errorf("field %s not read-only", formFields[i].key)
			}
		case *ui.SelectField:
			if !f.ReadOnly() {
				t.Errorf("field %s not read-only", formFields[i].key)
			}
		case *ui.CheckField:
			if !f.ReadOnly() {
				t.Errorf("field %s not read-only", formFields[i].key)
			}
		}
	}
}

func TestEnvVarNamesShownInView(t *testing.T) {
	cf := NewForm(Profile{Name: "env", FromEnv: true}, 0)
	view := cf.View()
	if !strings.Contains(view, "EMQUTITI_ENV_HOST") {
		t.Fatalf("expected env host var in view")
	}
	if !strings.Contains(view, "EMQUTITI_ENV_PORT") {
		t.Fatalf("expected env port var in view")
	}
}

func TestNewConnectionFormPasswordPlaceholder(t *testing.T) {
	cf := NewForm(Profile{Name: "test", Username: "user", Password: "secret"}, 0)
	tf, ok := cf.Fields[fieldIndex["Password"]].(*ui.TextField)
	if !ok {
		t.Fatalf("password field not text")
	}
	if tf.Model.Placeholder != "keyring:emqutiti-test/user" {
		t.Fatalf("unexpected placeholder %s", tf.Model.Placeholder)
	}
}

func TestConnectionFormProfile(t *testing.T) {
	cf := NewForm(Profile{}, -1)
	cf.Fields[fieldIndex["Name"]].(*ui.TextField).SetValue("n1")
	cf.Fields[fieldIndex["Port"]].(*ui.TextField).SetValue("1883")
	cf.Fields[fieldIndex["AutoReconnect"]].(*ui.CheckField).SetBool(true)
	cf.Fields[fieldIndex["QoS"]].(*ui.SelectField).Index = 2

	p, err := cf.Profile()
	if err != nil {
		t.Fatalf("Profile error: %v", err)
	}
	if p.Name != "n1" || p.Port != 1883 || !p.AutoReconnect || p.QoS != 2 {
		t.Fatalf("unexpected profile: %#v", p)
	}
}

func TestConnectionFormProfileInvalidInt(t *testing.T) {
	cf := NewForm(Profile{}, -1)
	cf.Fields[fieldIndex["Port"]].(*ui.TextField).SetValue("abc")
	p, err := cf.Profile()
	if err == nil {
		t.Fatalf("expected error for invalid port")
	}
	if p.Port != 0 {
		t.Fatalf("expected port 0, got %d", p.Port)
	}
}

func TestConnectionFormSchemaOptions(t *testing.T) {
	cf := NewForm(Profile{Schema: "mqtt"}, -1)
	sf := cf.Fields[fieldIndex["Schema"]].(*ui.SelectField)
	if sf.Value() != "mqtt" {
		t.Fatalf("expected schema mqtt, got %s", sf.Value())
	}
	cf = NewForm(Profile{Schema: "mqtts"}, -1)
	sf = cf.Fields[fieldIndex["Schema"]].(*ui.SelectField)
	if sf.Value() != "mqtts" {
		t.Fatalf("expected schema mqtts, got %s", sf.Value())
	}
}
