package pagination

import "strings"

func hasAllowedFilterOperatorSuffix(key string) bool {
	idx := strings.LastIndex(key, "_")
	if idx <= 0 || idx == len(key)-1 {
		return false
	}

	_, allowed := allowedFilterOperators[key[idx+1:]]
	return allowed
}

func parseFilterKey(key string) ([]string, string, string, error) {
	idx := strings.LastIndex(key, "_")
	if idx <= 0 || idx == len(key)-1 {
		return nil, "", "", ErrInvalidFilter
	}

	rawFields := key[:idx]
	operator := key[idx+1:]
	if rawFields == "" || operator == "" {
		return nil, "", "", ErrInvalidFilter
	}
	if _, allowed := allowedFilterOperators[operator]; !allowed {
		return nil, "", "", ErrInvalidFilter
	}

	segments := strings.Split(rawFields, ",")
	fields := make([]string, 0, len(segments))
	for _, segment := range segments {
		identifier, err := requireIdent(strings.TrimSpace(segment), ErrInvalidFilter)
		if err != nil {
			return nil, "", "", err
		}
		fields = append(fields, identifier)
	}

	if len(fields) == 0 {
		return nil, "", "", ErrInvalidFilter
	}

	normalized := strings.Join(fields, ",") + "_" + operator
	return fields, operator, normalized, nil
}
