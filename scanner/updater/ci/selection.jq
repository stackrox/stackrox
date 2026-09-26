# Use identities, not descriptions: Ubuntu decorates CVE names and OSV uses aliases.
def validate_record:
  if (type == "object" and (.Kind == "vulnerability" or .Kind == "enrichment") and
      (.Updater | type == "string") and (.Fingerprint | type == "string") and
      (.Ref | type == "string") and (.Date | type == "string") and
      ((.Kind == "vulnerability" and (.Vuln | type == "object") and (.Vuln.package.name | type == "string")) or
       (.Kind == "enrichment" and (.Enrichment | type == "object") and
        (((.Enrichment.Enrichment | type == "object") and (.Enrichment.Tags | type == "array")) or
         (.Enrichment.Enrichment == null and .Enrichment.Tags == null)))))
  then . else error("invalid importer record") end;

def vulnerability_ids:
  .Vuln |
  [(.name // ""), (.links // ""),
   ((.Self // {}) | ((.space // "") + "-" + (.name // ""))),
   ((.Aliases // [])[] | ((.space // "") + "-" + (.name // "")))] |
  .[] | scan("CVE-[0-9]{4}-[0-9]+|RH[BS]A-[0-9]{4}:[0-9]+|ALAS[0-9]*-[0-9]{4}-[0-9]+|GO-[0-9]{4}-[0-9]+|GHSA-[a-z0-9-]+");

def selected_vulnerability($ids):
  .Vuln as $v |
  ($ids[$v.name] // false) or
  ($ids[($v.name | split(" ")[0])] // false) or
  any(($v.Aliases // [])[]; $ids[(.space + "-" + .name)] // false) or
  any(($v.links // "" | scan("CVE-[0-9]{4}-[0-9]+|RH[BS]A-[0-9]{4}:[0-9]+|ALAS[0-9]*-[0-9]{4}-[0-9]+")); $ids[.] // false);

def selected_package($packages):
  .Vuln as $v |
  ($packages.distributions[((($v.distribution.did // "") + "/" + ($v.distribution.version_id // "")))][$v.package.name] // false) or
  ($packages.repositories[$v.repository.name // ""][$v.package.name] // false);

def select_record($selection; $test_cves; $ids; $packages; $alpine_packages; $did; $version):
  if $selection == "enrichment" then
    select(.Kind == "enrichment") |
    select(any((.Enrichment.Tags // [])[]; $ids[.] // false))
  else
    select(.Kind == "vulnerability" and .Vuln.package.name != "") |
    select(
      $selection == "all" or
      selected_vulnerability($test_cves) or
      ($selection == "packages" and selected_package($packages)) or
      ($selection == "alpine" and .Vuln.distribution.did == $did and
       .Vuln.distribution.version_id == $version and ($alpine_packages[.Vuln.package.name] // false)))
  end;
