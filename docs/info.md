# Showing information

## Show the hierarchical list of sub-namespaces

Use `kubectl accurate list`:

```console
$ kubectl accurate list
 root1
 root2
    ⮡sub1
 root3
 subroot1
    ⮡sn1
 subroot2
```

## Show all template Namespaces

Use `kubectl get ns -l accurate.cybozu.com/type=template`:

```console
$ kubectl get ns -l accurate.cybozu.com/type=template
NAME    STATUS   AGE
tmpl1   Active   3m45s
tmpl2   Active   3m43s
tmpl3   Active   3m43s
```

## Show the properties of a Namespace

Use `kubectl accurate ns describe`:

```console
$ kubectl accurate ns describe root2
Name: root2
Type: root
# of children: 1

Resources:
Kind     Name     From     Mode
-------- -------- -------- --------
Role     role1    tmpl3    create
Secret   mysecret          create
```
