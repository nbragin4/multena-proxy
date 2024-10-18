package main

type HashableFilter map[string]map[string]bool;

type Filter map[string][]string;
func (c Filter) AsMap() (HashableFilter) {
    hf := make(HashableFilter)

    // Convert Filter to HashableFilter
    for key, values := range c {
        innerMap := make(map[string]bool)
        for _, v := range values {
            innerMap[v] = true // Set bool value to true
        }
        hf[key] = innerMap
    }

    return hf
}

type LabelType = map[string]bool
type LabelConfigType = map[string]LabelType
