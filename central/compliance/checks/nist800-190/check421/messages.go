package check421

const interpretationText = `Docker only allows connecting to registries via an insecure protocol if this is explicitly specified in either the registry config 
or if the registry's IP is within an explicitly defined CIDR block. StackRox checks that insecure registries are only allowed
on a CIDR basis, and ensures that these CIDR blocks are within private subnets (such as 127.0.0.0/8 or 10.0.0.0/8).`
