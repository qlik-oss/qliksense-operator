// +build !ignore_autogenerated

// Code generated by operator-sdk. DO NOT EDIT.

package v1

import (
	status "github.com/operator-framework/operator-sdk/pkg/status"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Qliksense) DeepCopyInto(out *Qliksense) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Spec != nil {
		in, out := &in.Spec, &out.Spec
		*out = (*in).DeepCopy()
	}
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Qliksense.
func (in *Qliksense) DeepCopy() *Qliksense {
	if in == nil {
		return nil
	}
	out := new(Qliksense)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Qliksense) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QliksenseList) DeepCopyInto(out *QliksenseList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Qliksense, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QliksenseList.
func (in *QliksenseList) DeepCopy() *QliksenseList {
	if in == nil {
		return nil
	}
	out := new(QliksenseList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *QliksenseList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QliksenseSpec) DeepCopyInto(out *QliksenseSpec) {
	*out = *in
	in.CRSpec.DeepCopyInto(&out.CRSpec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QliksenseSpec.
func (in *QliksenseSpec) DeepCopy() *QliksenseSpec {
	if in == nil {
		return nil
	}
	out := new(QliksenseSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QliksenseStatus) DeepCopyInto(out *QliksenseStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make(status.Conditions, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QliksenseStatus.
func (in *QliksenseStatus) DeepCopy() *QliksenseStatus {
	if in == nil {
		return nil
	}
	out := new(QliksenseStatus)
	in.DeepCopyInto(out)
	return out
}
