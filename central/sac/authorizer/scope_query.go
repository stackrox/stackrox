package authorizer

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/observe"
	"github.com/stackrox/rox/pkg/uuid"
	"k8s.io/apimachinery/pkg/labels"
)

type scopeQueryBuilder struct {
	args []any
}

func (b *scopeQueryBuilder) bind(value any) string {
	b.args = append(b.args, value)
	return fmt.Sprintf("$%d", len(b.args))
}

func canonicalScopeID(id string) bool {
	parsed, err := uuid.FromString(id)
	return err == nil && parsed.String() == id
}

func (b *scopeQueryBuilder) clusterPredicate(rules *storage.SimpleAccessScope_Rules) (string, error) {
	var alternatives []string
	var ids []string
	for _, id := range rules.GetIncludedClusterIds() {
		if canonicalScopeID(id) {
			ids = append(ids, id)
		}
	}
	if len(ids) > 0 {
		alternatives = append(alternatives, "c.id = ANY("+b.bind(ids)+"::uuid[])")
	}
	if names := rules.GetIncludedClusters(); len(names) > 0 {
		alternatives = append(alternatives, "c.name = ANY("+b.bind(names)+"::text[])")
	}
	selectors, err := b.labelPredicates(rules.GetClusterLabelSelectors(), "c.labels", effectiveaccessscope.ClusterNameLabel, "COALESCE(c.name, '')", false)
	if err != nil {
		return "", errors.Wrap(err, "bad cluster label selector")
	}
	return scopeDisjunction(append(alternatives, selectors...)), nil
}

func (b *scopeQueryBuilder) namespacePredicate(rules *storage.SimpleAccessScope_Rules, exclude bool) (string, error) {
	var alternatives []string
	byID := make(map[string][]string)
	byName := make(map[string][]string)
	for _, namespace := range rules.GetIncludedNamespaces() {
		if namespace.GetNamespaceName() == "" {
			continue
		}
		if id := namespace.GetClusterId(); id != "" {
			// Match the existing evaluator: an ID takes precedence over a name,
			// even when the ID does not identify a known cluster.
			if !canonicalScopeID(id) {
				continue
			}
			byID[id] = append(byID[id], namespace.GetNamespaceName())
		} else if name := namespace.GetClusterName(); name != "" {
			byName[name] = append(byName[name], namespace.GetNamespaceName())
		}
	}
	// Bind namespace lists once per parent, rather than one pair of parameters
	// per namespace. This also keeps large named scopes within PostgreSQL's
	// parameter limit and lets the existing name/cluster-ID index serve them.
	for _, id := range slices.Sorted(maps.Keys(byID)) {
		alternatives = append(alternatives, "(n.clusterid = "+b.bind(id)+"::uuid AND n.name = ANY("+b.bind(byID[id])+"::text[]))")
	}
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		alternatives = append(alternatives, "(n.clustername = "+b.bind(name)+" AND n.name = ANY("+b.bind(byName[name])+"::text[]))")
	}
	if exclude {
		for i, alternative := range alternatives {
			alternatives[i] = "NOT COALESCE(" + alternative + ", FALSE)"
		}
	}
	selectors, err := b.labelPredicates(rules.GetNamespaceLabelSelectors(), "n.labels", effectiveaccessscope.NamespaceNameLabel,
		"COALESCE(n.clustername, '') || '::' || COALESCE(n.name, '')", exclude)
	if err != nil {
		return "", errors.Wrap(err, "bad namespace label selector")
	}
	if exclude {
		return scopeConjunction(append(alternatives, selectors...)), nil
	}
	return scopeDisjunction(append(alternatives, selectors...)), nil
}

func scopeConjunction(requirements []string) string {
	if len(requirements) == 0 {
		return "TRUE"
	}
	return "(" + strings.Join(requirements, " AND ") + ")"
}

func scopeDisjunction(alternatives []string) string {
	if len(alternatives) == 0 {
		return "FALSE"
	}
	return "(" + strings.Join(alternatives, " OR ") + ")"
}

func (b *scopeQueryBuilder) labelPredicates(selectors []*storage.SetBasedLabelSelector,
	column, virtualKey, virtualValue string, exclude bool,
) ([]string, error) {
	var alternatives []string
	for _, selector := range selectors {
		if len(selector.GetRequirements()) == 0 {
			// Empty selectors match nothing, rather than all objects.
			continue
		}
		var requirements []string
		for _, requirement := range selector.GetRequirements() {
			if _, err := labels.NewRequirement(requirement.GetKey(),
				effectiveaccessscope.ConvertLabelSelectorOperatorToSelectionOperator(requirement.GetOp()), requirement.GetValues()); err != nil {
				return nil, err
			}
			op := requirement.GetOp()
			if exclude {
				// Apply De Morgan's law to the grant predicate. In particular,
				// NOT_IN exclusions become positive containment predicates that
				// preserve the handling of missing labels.
				switch op {
				case storage.SetBasedLabelSelector_IN:
					op = storage.SetBasedLabelSelector_NOT_IN
				case storage.SetBasedLabelSelector_NOT_IN:
					op = storage.SetBasedLabelSelector_IN
				case storage.SetBasedLabelSelector_EXISTS:
					op = storage.SetBasedLabelSelector_NOT_EXISTS
				case storage.SetBasedLabelSelector_NOT_EXISTS:
					op = storage.SetBasedLabelSelector_EXISTS
				}
			}
			if requirement.GetKey() == virtualKey {
				// Virtual names overwrite stored labels and are always present.
				switch op {
				case storage.SetBasedLabelSelector_IN:
					requirements = append(requirements, virtualValue+" = ANY("+b.bind(requirement.GetValues())+"::text[])")
				case storage.SetBasedLabelSelector_NOT_IN:
					requirements = append(requirements, virtualValue+" <> ALL("+b.bind(requirement.GetValues())+"::text[])")
				case storage.SetBasedLabelSelector_EXISTS:
					requirements = append(requirements, "TRUE")
				case storage.SetBasedLabelSelector_NOT_EXISTS:
					requirements = append(requirements, "FALSE")
				}
				continue
			}
			switch op {
			case storage.SetBasedLabelSelector_IN, storage.SetBasedLabelSelector_NOT_IN:
				var values []string
				for _, value := range requirement.GetValues() {
					values = append(values, column+" @> "+b.bind(map[string]string{requirement.GetKey(): value})+"::jsonb")
				}
				predicate := scopeDisjunction(values)
				if op == storage.SetBasedLabelSelector_NOT_IN {
					predicate = "NOT COALESCE(" + predicate + ", FALSE)"
				}
				requirements = append(requirements, predicate)
			case storage.SetBasedLabelSelector_EXISTS:
				requirements = append(requirements, column+" ? "+b.bind(requirement.GetKey()))
			case storage.SetBasedLabelSelector_NOT_EXISTS:
				requirements = append(requirements, "NOT COALESCE("+column+" ? "+b.bind(requirement.GetKey())+", FALSE)")
			}
		}
		if exclude {
			alternatives = append(alternatives, scopeDisjunction(requirements))
		} else {
			alternatives = append(alternatives, scopeConjunction(requirements))
		}
	}
	return alternatives, nil
}

func scopeQuery(rules *storage.SimpleAccessScope_Rules) (string, []any, error) {
	var builder scopeQueryBuilder
	clusters, err := builder.clusterPredicate(rules)
	if err != nil {
		return "", nil, err
	}
	exclusions := hasUnboundedNamespaceSelector(rules)
	namespaces, err := builder.namespacePredicate(rules, exclusions)
	if err != nil {
		return "", nil, err
	}
	if clusters == "FALSE" && ((!exclusions && namespaces == "FALSE") || (exclusions && namespaces == "TRUE")) {
		return "", nil, nil
	}
	if exclusions {
		// Pure negative selectors can grant most namespaces. Return exclusions
		// plus a compact inventory of names/IDs, instead of allocating an
		// Included tree node for every match. Sparse groups keep the IN form.
		parentFilter := "TRUE"
		if clusters != "FALSE" {
			parentFilter = "NOT EXISTS (SELECT 1 FROM included_clusters i WHERE i.name = n.clustername)"
		}
		query := `WITH included_clusters AS (
			SELECT c.id, c.name FROM clusters c WHERE ` + clusters + `
		), namespace_groups AS (
			SELECT n.clustername,
				array_agg(n.id::text || COALESCE(n.name, '')) FILTER (WHERE NOT COALESCE(` + namespaces + `, FALSE)) AS matched,
				array_agg(n.id::text || COALESCE(n.name, '')) FILTER (WHERE ` + namespaces + `) AS unmatched
			FROM namespaces n WHERE ` + parentFilter + `
			GROUP BY n.clustername
		)
		SELECT c.id, c.name, NULL::text, NULL::text[], NULL::text[]
		FROM included_clusters c
		UNION ALL
		SELECT c.id, c.name, g.clustername, g.matched,
			CASE WHEN cardinality(matched) > COALESCE(cardinality(unmatched), 0)
				THEN COALESCE(unmatched, ARRAY[]::text[]) ELSE NULL::text[] END
		FROM namespace_groups g LEFT JOIN clusters c ON c.name = g.clustername
		WHERE cardinality(matched) > 0`
		return query, builder.args, nil
	}
	// The existing evaluator groups namespaces by their recorded cluster name.
	// Preserve that relationship, including orphaned or inconsistent metadata,
	// rather than replacing it with a join on cluster ID.
	query := `WITH included_clusters AS (
		SELECT c.id, c.name FROM clusters c WHERE ` + clusters + `
	)
	SELECT c.id, c.name, NULL::text, NULL::text[], NULL::text[]
	FROM included_clusters c
	UNION ALL
	SELECT c.id, c.name, n.clustername,
		array_agg(n.id::text || COALESCE(n.name, '')), NULL::text[]
	FROM namespaces n LEFT JOIN clusters c ON c.name = n.clustername
	WHERE ` + namespaces + `
	AND NOT EXISTS (SELECT 1 FROM included_clusters i WHERE i.name = n.clustername)
	GROUP BY c.id, c.name, n.clustername`
	return query, builder.args, nil
}

// A selector bounded by IN or EXISTS can use the indexed matching query. A
// pure NOT_IN/NOT_EXISTS alternative may match most of the inventory, even when
// other alternatives are positive. Requirements still AND and alternatives OR.
func hasUnboundedNamespaceSelector(rules *storage.SimpleAccessScope_Rules) bool {
	for _, selector := range rules.GetNamespaceLabelSelectors() {
		if len(selector.GetRequirements()) == 0 {
			continue
		}
		negative := true
		for _, requirement := range selector.GetRequirements() {
			if requirement.GetOp() != storage.SetBasedLabelSelector_NOT_IN && requirement.GetOp() != storage.SetBasedLabelSelector_NOT_EXISTS {
				negative = false
				break
			}
		}
		if negative {
			return true
		}
	}
	return false
}

func computeSQLScope(ctx context.Context, db postgres.DB, rules *storage.SimpleAccessScope_Rules) (*effectiveaccessscope.ScopeTree, error) {
	query, args, err := scopeQuery(rules)
	if err != nil {
		return nil, err
	}
	if query == "" {
		return effectiveaccessscope.DenyAllEffectiveAccessScope(), nil
	}
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "querying access scope metadata")
	}
	defer rows.Close()
	var clusters []effectiveaccessscope.Cluster
	var namespaces []effectiveaccessscope.MatchedNamespaceGroup
	var includedClusterIDs []string
	seenClusters := make(map[string]bool)
	for rows.Next() {
		var clusterID, clusterName, namespaceClusterName *string
		var namespaceMatches, namespaceExclusions []string
		if err := rows.Scan(&clusterID, &clusterName, &namespaceClusterName, &namespaceMatches, &namespaceExclusions); err != nil {
			return nil, errors.Wrap(err, "reading access scope metadata")
		}
		if clusterID != nil && !seenClusters[*clusterID] {
			clusters = append(clusters, &storage.Cluster{Id: *clusterID, Name: scopeString(clusterName)})
			seenClusters[*clusterID] = true
		}
		if len(namespaceMatches) > 0 {
			// UUIDs have a fixed-width text representation. Return each ID/name
			// pair as one array element, so pairing needs neither two array
			// decoders nor SQL ORDER BY. Names can contain arbitrary characters.
			group := effectiveaccessscope.MatchedNamespaceGroup{
				ClusterName: scopeString(namespaceClusterName), Complete: namespaceExclusions != nil,
				Namespaces:         make([]effectiveaccessscope.MatchedNamespace, len(namespaceMatches)),
				ExcludedNamespaces: make([]effectiveaccessscope.MatchedNamespace, len(namespaceExclusions)),
			}
			for i, entries := range [][]string{namespaceMatches, namespaceExclusions} {
				for j, match := range entries {
					const idLength = len("00000000-0000-0000-0000-000000000000")
					if len(match) < idLength {
						return nil, errors.New("invalid namespace metadata from access scope query")
					}
					metadata := effectiveaccessscope.MatchedNamespace{ID: match[:idLength], Name: match[idLength:]}
					if i == 0 {
						group.Namespaces[j] = metadata
					} else {
						group.ExcludedNamespaces[j] = metadata
					}
				}
			}
			namespaces = append(namespaces, group)
		} else if clusterID != nil {
			includedClusterIDs = append(includedClusterIDs, *clusterID)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "reading access scope metadata")
	}
	// Build the same tree from matched IDs and names. Labels and serialized
	// objects stay in PostgreSQL; no inventory-wide matching runs in Go.
	return effectiveaccessscope.FromMatchingScopes(clusters, includedClusterIDs, namespaces), nil
}

func scopeString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func recordScopeInventory(ctx context.Context, db postgres.DB, trace *observe.AuthzTrace) error {
	rows, err := db.Query(ctx, "SELECT id, name FROM clusters")
	if err != nil {
		return errors.Wrap(err, "reading cluster names for authorization tracing")
	}
	defer rows.Close()
	clusterNames := make(map[string]string)
	for rows.Next() {
		var id string
		var name *string
		if err := rows.Scan(&id, &name); err != nil {
			return errors.Wrap(err, "reading cluster names for authorization tracing")
		}
		clusterNames[id] = scopeString(name)
	}
	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "reading cluster names for authorization tracing")
	}
	rows.Close()
	var namespaceCount int32
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM namespaces").Scan(&namespaceCount); err != nil {
		return errors.Wrap(err, "counting namespaces for authorization tracing")
	}
	trace.RecordKnownClusterNames(clusterNames, namespaceCount)
	return nil
}
