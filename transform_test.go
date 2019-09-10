package manifestival_test

import (
	"os"
	"testing"

	. "github.com/jcrossley3/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"
)

func TestTransform(t *testing.T) {
	m, _ := NewManifest("testdata/tree", true, &rest.Config{})
	f := &m
	actual := f.Resources
	if len(actual) != 5 {
		t.Errorf("Failed to read all resources: %s", actual)
	}
	f, _ = f.Transform(func(u *unstructured.Unstructured) error {
		if u.GetKind() == "B" {
			u.SetResourceVersion("69")
		}
		return nil
	})
	transformed := f.Resources
	// Ensure all transformed have a version and B kind
	for _, spec := range transformed {
		if spec.GetResourceVersion() != "69" && spec.GetKind() == "B" {
			t.Errorf("The transform didn't work: %s", transformed)
		}
	}
	// Ensure we didn't change the previous resources
	for _, spec := range actual {
		if spec.GetResourceVersion() != "" {
			t.Errorf("The transform shouldn't affect previous resources: %s", actual)
		}
	}
}

func TestTransformCombo(t *testing.T) {
	m, err := NewManifest("testdata/tree", true, &rest.Config{})
	f := &m
	if len(f.Resources) != 5 {
		t.Errorf("Failed to read all resources: %s", f.Resources)
	}
	fn1 := func(u *unstructured.Unstructured) error {
		if u.GetKind() == "B" {
			u.SetResourceVersion("69")
		}
		return nil
	}
	fn2 := func(u *unstructured.Unstructured) error {
		if u.GetName() == "bar" {
			u.SetResourceVersion("42")
		}
		return nil
	}
	if f, err = f.Transform(fn1, fn2); err != nil {
		t.Error(err)
	}
	for _, x := range f.Resources {
		if x.GetName() == "bar" && x.GetResourceVersion() != "42" {
			t.Errorf("Failed to transform bar by combo: %s", x)
		}
		if x.GetName() == "B" && x.GetResourceVersion() != "69" {
			t.Errorf("Failed to transform B by combo: %s", x)
		}
	}
}

func TestInjectNamespace(t *testing.T) {
	assert := func(u unstructured.Unstructured, expected string) {
		v, _, _ := unstructured.NestedSlice(u.Object, "subjects")
		ns := v[0].(map[string]interface{})["namespace"]
		if ns != expected {
			t.Errorf("Expected '%s', got '%s'", expected, ns)
		}
	}
	m, err := NewManifest("testdata/crb.yaml", true, &rest.Config{})
	f := &m
	if len(f.Resources) != 2 {
		t.Errorf("Expected 2 resources from crb.yaml, got %d (%s)", len(f.Resources), err)
	}
	if f, err = f.Transform(InjectNamespace("foo")); err != nil {
		t.Error(err)
	}
	if len(f.Resources) != 2 {
		t.Errorf("Expected 2 resources with 'foo' ns, got %d", len(f.Resources))
	}
	if f.Resources[0].GetName() != "foo" {
		t.Errorf("Expected namespace name to be foo, got %s", f.Resources[0].GetName())
	}
	assert(f.Resources[1], "foo")
	os.Setenv("FOO", "food")
	if f, err = f.Transform(InjectNamespace("$FOO")); err != nil {
		t.Error(err)
	}
	if f.Resources[0].GetName() != "food" {
		t.Errorf("Expected namespace name to be food, got %s", f.Resources[0].GetName())
	}
	assert(f.Resources[1], "food")
}
