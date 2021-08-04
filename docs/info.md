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

Use `kubectl accurate template list`:

```console
$ kubectl accurate template list
 template1
 template2
    ⮡reference1
    ⮡reference2
 template3
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
