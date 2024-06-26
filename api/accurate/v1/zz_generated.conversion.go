//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by conversion-gen. DO NOT EDIT.

package v1

import (
	unsafe "unsafe"

	v2 "github.com/cybozu-go/accurate/api/accurate/v2"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*SubNamespaceList)(nil), (*v2.SubNamespaceList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1_SubNamespaceList_To_v2_SubNamespaceList(a.(*SubNamespaceList), b.(*v2.SubNamespaceList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v2.SubNamespaceList)(nil), (*SubNamespaceList)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v2_SubNamespaceList_To_v1_SubNamespaceList(a.(*v2.SubNamespaceList), b.(*SubNamespaceList), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*SubNamespaceSpec)(nil), (*v2.SubNamespaceSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1_SubNamespaceSpec_To_v2_SubNamespaceSpec(a.(*SubNamespaceSpec), b.(*v2.SubNamespaceSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*v2.SubNamespaceSpec)(nil), (*SubNamespaceSpec)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v2_SubNamespaceSpec_To_v1_SubNamespaceSpec(a.(*v2.SubNamespaceSpec), b.(*SubNamespaceSpec), scope)
	}); err != nil {
		return err
	}
	if err := s.AddConversionFunc((*SubNamespace)(nil), (*v2.SubNamespace)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1_SubNamespace_To_v2_SubNamespace(a.(*SubNamespace), b.(*v2.SubNamespace), scope)
	}); err != nil {
		return err
	}
	if err := s.AddConversionFunc((*v2.SubNamespace)(nil), (*SubNamespace)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v2_SubNamespace_To_v1_SubNamespace(a.(*v2.SubNamespace), b.(*SubNamespace), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1_SubNamespace_To_v2_SubNamespace(in *SubNamespace, out *v2.SubNamespace, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1_SubNamespaceSpec_To_v2_SubNamespaceSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	// WARNING: in.Status requires manual conversion: inconvertible types (github.com/cybozu-go/accurate/api/accurate/v1.SubNamespaceStatus vs github.com/cybozu-go/accurate/api/accurate/v2.SubNamespaceStatus)
	return nil
}

func autoConvert_v2_SubNamespace_To_v1_SubNamespace(in *v2.SubNamespace, out *SubNamespace, s conversion.Scope) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v2_SubNamespaceSpec_To_v1_SubNamespaceSpec(&in.Spec, &out.Spec, s); err != nil {
		return err
	}
	// WARNING: in.Status requires manual conversion: inconvertible types (github.com/cybozu-go/accurate/api/accurate/v2.SubNamespaceStatus vs github.com/cybozu-go/accurate/api/accurate/v1.SubNamespaceStatus)
	return nil
}

func autoConvert_v1_SubNamespaceList_To_v2_SubNamespaceList(in *SubNamespaceList, out *v2.SubNamespaceList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]v2.SubNamespace, len(*in))
		for i := range *in {
			if err := Convert_v1_SubNamespace_To_v2_SubNamespace(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

// Convert_v1_SubNamespaceList_To_v2_SubNamespaceList is an autogenerated conversion function.
func Convert_v1_SubNamespaceList_To_v2_SubNamespaceList(in *SubNamespaceList, out *v2.SubNamespaceList, s conversion.Scope) error {
	return autoConvert_v1_SubNamespaceList_To_v2_SubNamespaceList(in, out, s)
}

func autoConvert_v2_SubNamespaceList_To_v1_SubNamespaceList(in *v2.SubNamespaceList, out *SubNamespaceList, s conversion.Scope) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]SubNamespace, len(*in))
		for i := range *in {
			if err := Convert_v2_SubNamespace_To_v1_SubNamespace(&(*in)[i], &(*out)[i], s); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

// Convert_v2_SubNamespaceList_To_v1_SubNamespaceList is an autogenerated conversion function.
func Convert_v2_SubNamespaceList_To_v1_SubNamespaceList(in *v2.SubNamespaceList, out *SubNamespaceList, s conversion.Scope) error {
	return autoConvert_v2_SubNamespaceList_To_v1_SubNamespaceList(in, out, s)
}

func autoConvert_v1_SubNamespaceSpec_To_v2_SubNamespaceSpec(in *SubNamespaceSpec, out *v2.SubNamespaceSpec, s conversion.Scope) error {
	out.Labels = *(*map[string]string)(unsafe.Pointer(&in.Labels))
	out.Annotations = *(*map[string]string)(unsafe.Pointer(&in.Annotations))
	return nil
}

// Convert_v1_SubNamespaceSpec_To_v2_SubNamespaceSpec is an autogenerated conversion function.
func Convert_v1_SubNamespaceSpec_To_v2_SubNamespaceSpec(in *SubNamespaceSpec, out *v2.SubNamespaceSpec, s conversion.Scope) error {
	return autoConvert_v1_SubNamespaceSpec_To_v2_SubNamespaceSpec(in, out, s)
}

func autoConvert_v2_SubNamespaceSpec_To_v1_SubNamespaceSpec(in *v2.SubNamespaceSpec, out *SubNamespaceSpec, s conversion.Scope) error {
	out.Labels = *(*map[string]string)(unsafe.Pointer(&in.Labels))
	out.Annotations = *(*map[string]string)(unsafe.Pointer(&in.Annotations))
	return nil
}

// Convert_v2_SubNamespaceSpec_To_v1_SubNamespaceSpec is an autogenerated conversion function.
func Convert_v2_SubNamespaceSpec_To_v1_SubNamespaceSpec(in *v2.SubNamespaceSpec, out *SubNamespaceSpec, s conversion.Scope) error {
	return autoConvert_v2_SubNamespaceSpec_To_v1_SubNamespaceSpec(in, out, s)
}
