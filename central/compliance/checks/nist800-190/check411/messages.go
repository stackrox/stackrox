package check411

const interpretationText = `StackRox allows you to integrate with image scanners. When new vulnerabilities are discovered,
image scanners will mark images affected with the CVE, provided a scanner is integrated. StackRox also provides many build 
and deploy time policies that can fail if the image has a critical vulnerability. Therefore, a cluster is compliant if 
there is an image scanner in use with policies to ensure images with critical vulnerabilities are not deployed and 
build-time policies are being enforced.`
