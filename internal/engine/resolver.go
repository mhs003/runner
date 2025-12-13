package engine

import "mhs003/runner/internal/config"

func Resolve(f *config.File, name string, seen map[string]bool, out *[]*config.Task) {
    if seen[name] {
        return
    }
    seen[name] = true

    t := f.Tasks[name]
    for _, d := range t.Deps {
        Resolve(f, d, seen, out)
    }

    *out = append(*out, t)
}