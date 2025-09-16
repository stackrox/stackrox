# `explain` analysis


The goal of `explain` analysis is to provide additional connectivity information, specifying the resources (such as network policies, admin network policies, routes and more) that contribute to allowing or denying a connection between any pair of input workloads.


The report can help testing whether the configured resources induce connectivity as expected, and give hints to where the resources may be changed to achieve the desired result.

The `explain` analysis is supported with `txt` output format only.


### Textual Output Example

Example run with `txt` output to `stdout`:
```shell
$ roxctl netpol connectivity map --explain roxctl/netpol/connectivity/map/testdata/minimal
```
```txt
# Specific connections and their reasons # 
----------------------------------------------------------------------------------------------------------------------------------------------------------------
Connections between 0.0.0.0-255.255.255.255 => default/backend[Deployment]:

Denied connections:
        Denied TCP, UDP, SCTP due to the following policies and rules:
                Egress (Allowed) due to the system default (Allow all)
                Ingress (Denied)
                        NetworkPolicy list:
                                - NetworkPolicy 'default/backend-netpol' selects default/backend[Deployment], but 0.0.0.0-255.255.255.255 is not allowed by any Ingress rule
                                - NetworkPolicy 'default/default-deny-in-namespace' selects default/backend[Deployment], but 0.0.0.0-255.255.255.255 is not allowed by any Ingress rule (no rules defined)


----------------------------------------------------------------------------------------------------------------------------------------------------------------
Connections between 0.0.0.0-255.255.255.255 => default/frontend[Deployment]:

Allowed connections:
        Allowed TCP:[8080] due to the following policies and rules:
                Egress (Allowed) due to the system default (Allow all)
                Ingress (Allowed)
                        NetworkPolicy 'default/frontend-netpol' allows connections by Ingress rule #1

Denied connections:
        Denied TCP:[1-8079,8081-65535], UDP, SCTP due to the following policies and rules:
                Egress (Allowed) due to the system default (Allow all)
                Ingress (Denied)
                        NetworkPolicy list:
                                - NetworkPolicy 'default/default-deny-in-namespace' selects default/frontend[Deployment], but 0.0.0.0-255.255.255.255 is not allowed by any Ingress rule (no rules defined)
                                - NetworkPolicy 'default/frontend-netpol' selects default/frontend[Deployment], and Ingress rule #1 selects 0.0.0.0-255.255.255.255, but the protocols and ports do not match


----------------------------------------------------------------------------------------------------------------------------------------------------------------
Connections between default/backend[Deployment] => 0.0.0.0-255.255.255.255:

Denied connections:
        Denied TCP, UDP, SCTP due to the following policies and rules:
                Egress (Denied)
                        NetworkPolicy list:
                                - NetworkPolicy 'default/backend-netpol' selects default/backend[Deployment], but 0.0.0.0-255.255.255.255 is not allowed by any Egress rule (no rules defined)
                                - NetworkPolicy 'default/default-deny-in-namespace' selects default/backend[Deployment], but 0.0.0.0-255.255.255.255 is not allowed by any Egress rule (no rules defined)

                Ingress (Allowed) due to the system default (Allow all)

----------------------------------------------------------------------------------------------------------------------------------------------------------------
Connections between default/backend[Deployment] => default/frontend[Deployment]:

Denied connections:
        Denied TCP:[1-8079,8081-65535], UDP, SCTP due to the following policies and rules:
                Egress (Denied)
                        NetworkPolicy list:
                                - NetworkPolicy 'default/backend-netpol' selects default/backend[Deployment], but default/frontend[Deployment] is not allowed by any Egress rule (no rules defined)
                                - NetworkPolicy 'default/default-deny-in-namespace' selects default/backend[Deployment], but default/frontend[Deployment] is not allowed by any Egress rule (no rules defined)

                Ingress (Denied)
                        NetworkPolicy list:
                                - NetworkPolicy 'default/default-deny-in-namespace' selects default/frontend[Deployment], but default/backend[Deployment] is not allowed by any Ingress rule (no rules defined)
                                - NetworkPolicy 'default/frontend-netpol' selects default/frontend[Deployment], and Ingress rule #1 selects default/backend[Deployment], but the protocols and ports do not match


        Denied TCP:[8080] due to the following policies and rules:
                Egress (Denied)
                        NetworkPolicy list:
                                - NetworkPolicy 'default/backend-netpol' selects default/backend[Deployment], but default/frontend[Deployment] is not allowed by any Egress rule (no rules defined)
                                - NetworkPolicy 'default/default-deny-in-namespace' selects default/backend[Deployment], but default/frontend[Deployment] is not allowed by any Egress rule (no rules defined)

                Ingress (Allowed)
                        NetworkPolicy 'default/frontend-netpol' allows connections by Ingress rule #1

----------------------------------------------------------------------------------------------------------------------------------------------------------------
Connections between default/frontend[Deployment] => 0.0.0.0-255.255.255.255:

Allowed connections:
        Allowed UDP:[53] due to the following policies and rules:
                Egress (Allowed)
                        NetworkPolicy 'default/frontend-netpol' allows connections by Egress rule #2
                Ingress (Allowed) due to the system default (Allow all)

Denied connections:
        Denied TCP, UDP:[1-52,54-65535], SCTP due to the following policies and rules:
                Egress (Denied)
                        NetworkPolicy list:
                                - NetworkPolicy 'default/default-deny-in-namespace' selects default/frontend[Deployment], but 0.0.0.0-255.255.255.255 is not allowed by any Egress rule (no rules defined)
                                - NetworkPolicy 'default/frontend-netpol' selects default/frontend[Deployment], and Egress rule #2 selects 0.0.0.0-255.255.255.255, but the protocols and ports do not match

                Ingress (Allowed) due to the system default (Allow all)

----------------------------------------------------------------------------------------------------------------------------------------------------------------
Connections between default/frontend[Deployment] => default/backend[Deployment]:

Allowed connections:
        Allowed TCP:[9090] due to the following policies and rules:
                Egress (Allowed)
                        NetworkPolicy 'default/frontend-netpol' allows connections by Egress rule #1
                Ingress (Allowed)
                        NetworkPolicy 'default/backend-netpol' allows connections by Ingress rule #1

Denied connections:
        Denied TCP:[1-9089,9091-65535], UDP:[1-52,54-65535], SCTP due to the following policies and rules:
                Egress (Denied)
                        NetworkPolicy list:
                                - NetworkPolicy 'default/default-deny-in-namespace' selects default/frontend[Deployment], but default/backend[Deployment] is not allowed by any Egress rule (no rules defined)
                                - NetworkPolicy 'default/frontend-netpol' selects default/frontend[Deployment], and Egress rule #1 selects default/backend[Deployment], but the protocols and ports do not match

                Ingress (Denied)
                        NetworkPolicy list:
                                - NetworkPolicy 'default/backend-netpol' selects default/backend[Deployment], and Ingress rule #1 selects default/frontend[Deployment], but the protocols and ports do not match
                                - NetworkPolicy 'default/default-deny-in-namespace' selects default/backend[Deployment], but default/frontend[Deployment] is not allowed by any Ingress rule (no rules defined)


        Denied UDP:[53] due to the following policies and rules:
                Egress (Allowed)
                        NetworkPolicy 'default/frontend-netpol' allows connections by Egress rule #2
                Ingress (Denied)
                        NetworkPolicy list:
                                - NetworkPolicy 'default/backend-netpol' selects default/backend[Deployment], and Ingress rule #1 selects default/frontend[Deployment], but the protocols and ports do not match
                                - NetworkPolicy 'default/default-deny-in-namespace' selects default/backend[Deployment], but default/frontend[Deployment] is not allowed by any Ingress rule (no rules defined)
```

#### Understanding the output

The results of `explain` analysis extend the simple `connectivity map` with explanations about the connections between pairs of input workloads.

1. Connections which are concluded from policies' rules are reported in a section `Specific connections and their reasons`;
in this section, for each `src` => `dst` pair :
        
- if there are allowed connections from `src` to `dst`, it reports under internal section `Allowed connections`; for each allowed connection which egress networking resource(s) selects the `src` and what rules allow the connection; and likewise the rules of `ingress` networking resource(s) capturing `dst` that allows the connection.
- it also specifies the networking resource(s) and reasons for the denied connections from `src` to `dst` under internal section `Denied connections` on both egress and ingress directions

2. `src`=>`dst` connections which are not captured by policies and derived from system-default "allow all" are grouped under a separate section: `All Connections due to the system default (Allow all)` (if found)
