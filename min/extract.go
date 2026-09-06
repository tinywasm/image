//go:build !wasm

package min

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	"webtyp.com/image"
)

func ExtractImages(moduleDir string) ([]ParsedAsset, error) {
	imagePath := filepath.Join(moduleDir, "image.go")
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, imagePath, nil, 0)
	if err != nil {
		return nil, nil
	}
	var assets []ParsedAsset
	ast.Inspect(node, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "RenderImages" {
			return true
		}
		ast.Inspect(fn.Body, func(bn ast.Node) bool {
			ret, ok := bn.(*ast.ReturnStmt)
			if !ok {
				return true
			}
			for _, res := range ret.Results {
				switch r := res.(type) {
				case *ast.CompositeLit:
					assets = parseAssetSlice(r, moduleDir)
				case *ast.Ident:
					// Look for variable definition in the same function
					ast.Inspect(fn.Body, func(vn ast.Node) bool {
						assign, ok := vn.(*ast.AssignStmt)
						if !ok {
							return true
						}
						for _, lhs := range assign.Lhs {
							if id, ok := lhs.(*ast.Ident); ok && id.Name == r.Name {
								if cl, ok := assign.Rhs[0].(*ast.CompositeLit); ok {
									assets = parseAssetSlice(cl, moduleDir)
								}
								return false
							}
						}
						return true
					})
				}
			}
			return false
		})
		return false
	})
	return assets, nil
}

func parseAssetSlice(cl *ast.CompositeLit, moduleDir string) []ParsedAsset {
	var assets []ParsedAsset
	for _, elt := range cl.Elts {
		if acl, ok := elt.(*ast.CompositeLit); ok {
			asset := parseAsset(acl, moduleDir)
			if asset.BaseName != "" {
				assets = append(assets, asset)
			}
		}
	}
	return assets
}

func parseAsset(cl *ast.CompositeLit, moduleDir string) ParsedAsset {
	var asset ParsedAsset
	var path string
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key := fmt.Sprintf("%s", kv.Key)
		switch key {
		case "Path":
			if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				p, _ := strconv.Unquote(lit.Value)
				path = p
			}
		case "Variants":
			asset.Variants = resolveVariants(kv.Value)
		case "Alt":
			if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
				asset.Alt, _ = strconv.Unquote(lit.Value)
			}
		}
	}
	if path != "" {
		asset.AbsPath = filepath.Join(moduleDir, path)
		base := filepath.Base(path)
		ext := filepath.Ext(base)
		asset.BaseName = strings.TrimSuffix(base, ext)
		if asset.Alt == "" {
			asset.Alt = deriveAlt(asset.BaseName)
		}
	}
	return asset
}

func resolveVariants(expr ast.Expr) image.Variant {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		return variantFromName(e.Sel.Name)
	case *ast.BinaryExpr:
		if e.Op == token.OR || e.Op == token.LOR {
			return resolveVariants(e.X) | resolveVariants(e.Y)
		}
	case *ast.Ident:
		return variantFromName(e.Name)
	case *ast.CallExpr:
		if len(e.Args) == 1 {
			return resolveVariants(e.Args[0])
		}
	case *ast.BasicLit:
		if e.Kind == token.INT {
			v, _ := strconv.ParseUint(e.Value, 10, 8)
			return image.Variant(v)
		}
	}
	return 0
}

func variantFromName(name string) image.Variant {
	switch name {
	case "VariantS":
		return image.VariantS
	case "VariantM":
		return image.VariantM
	case "VariantL":
		return image.VariantL
	case "AllVariants":
		return image.AllVariants
	default:
		return 0
	}
}
