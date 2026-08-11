package eeexpr

import "fmt"

// Typed wrappers over Invoke. Every function name and argument key below was
// read out of a captured fixture in testdata/ — none was guessed.
//
// Where the PRD's illustrative examples disagree with the fixtures, the
// fixtures win. Notable corrections (all verified, see docs/KNOWN_ISSUES.md):
//
//	PRD §5.3  Collection.filterDate{collection,start,end}
//	actual    Collection.filter + Filter.dateRangeContains
//	PRD §11.1 Image.rename{bandNames}
//	actual    Image.rename{input, names}
//	PRD §11.1 Image.divide(scalar)
//	actual    Image.divide{image1, image2}, scalar wrapped in Image.constant

// ── Image ───────────────────────────────────────────────────────────────────

// Image wraps a Node known to be an ee.Image.
type Image struct{ N Node }

func Img(n Node) Image { return Image{N: n} }

// ImageConstant is ee.Image(<number>). Scalar operands to arithmetic are always
// promoted through this — Earth Engine has no scalar overloads.
func ImageConstant(v float64) Image {
	return Image{Invoke("Image.constant", map[string]Node{"value": Const(v)})}
}

// ImageLoad is ee.Image("<asset id>").
func ImageLoad(id string) Image {
	return Image{Invoke("Image.load", map[string]Node{"id": Const(id)})}
}

func (i Image) Select(bands ...string) Image {
	return Image{Invoke("Image.select", map[string]Node{
		"input":         i.N,
		"bandSelectors": Const(bands),
	})}
}

func (i Image) Rename(names ...string) Image {
	return Image{Invoke("Image.rename", map[string]Node{
		"input": i.N,
		"names": Const(names),
	})}
}

// NormalizedDifference is (a - b) / (a + b) over two band names.
func (i Image) NormalizedDifference(a, b string) Image {
	return Image{Invoke("Image.normalizedDifference", map[string]Node{
		"input":     i.N,
		"bandNames": Const([]string{a, b}),
	})}
}

func (i Image) binary(fn string, other Image) Image {
	return Image{Invoke(fn, map[string]Node{"image1": i.N, "image2": other.N})}
}

func (i Image) Add(o Image) Image      { return i.binary("Image.add", o) }
func (i Image) Subtract(o Image) Image { return i.binary("Image.subtract", o) }
func (i Image) Multiply(o Image) Image { return i.binary("Image.multiply", o) }
func (i Image) Divide(o Image) Image   { return i.binary("Image.divide", o) }
func (i Image) And(o Image) Image      { return i.binary("Image.and", o) }
func (i Image) Neq(o Image) Image      { return i.binary("Image.neq", o) }
func (i Image) Gte(o Image) Image      { return i.binary("Image.gte", o) }
func (i Image) Max(o Image) Image      { return i.binary("Image.max", o) }
func (i Image) Min(o Image) Image      { return i.binary("Image.min", o) }
func (i Image) Pow(o Image) Image      { return i.binary("Image.pow", o) }

// AddNum and the other *Num forms take a scalar operand, promoting it through
// Image.constant — exactly what the Python client does, since Earth Engine has
// no scalar arithmetic overloads.
func (i Image) AddNum(v float64) Image      { return i.Add(ImageConstant(v)) }
func (i Image) SubtractNum(v float64) Image { return i.Subtract(ImageConstant(v)) }
func (i Image) MultiplyNum(v float64) Image { return i.Multiply(ImageConstant(v)) }
func (i Image) DivideNum(v float64) Image   { return i.Divide(ImageConstant(v)) }
func (i Image) NeqNum(v float64) Image      { return i.Neq(ImageConstant(v)) }
func (i Image) GteNum(v float64) Image      { return i.Gte(ImageConstant(v)) }
func (i Image) MaxNum(v float64) Image      { return i.Max(ImageConstant(v)) }
func (i Image) MinNum(v float64) Image      { return i.Min(ImageConstant(v)) }

func (i Image) UpdateMask(mask Image) Image {
	return Image{Invoke("Image.updateMask", map[string]Node{
		"image": i.N, "mask": mask.N,
	})}
}

func (i Image) Clamp(lo, hi float64) Image {
	return Image{Invoke("Image.clamp", map[string]Node{
		"input": i.N, "low": Const(lo), "high": Const(hi),
	})}
}

func (i Image) Floor() Image {
	return Image{Invoke("Image.floor", map[string]Node{"value": i.N})}
}

func (i Image) Int() Image {
	return Image{Invoke("Image.int", map[string]Node{"value": i.N})}
}

// AddBands appends srcImg's bands. names==nil means "all bands".
func (i Image) AddBands(src Image) Image {
	return Image{Invoke("Image.addBands", map[string]Node{
		"dstImg": i.N, "srcImg": src.N,
	})}
}

func (i Image) Clip(geom Geometry) Image {
	return Image{Invoke("Image.clip", map[string]Node{
		"input": i.N, "geometry": geom.N,
	})}
}

func (i Image) Resample(mode string) Image {
	return Image{Invoke("Image.resample", map[string]Node{
		"image": i.N, "mode": Const(mode),
	})}
}

// Reproject wraps the CRS in a Projection node — the argument is a projection,
// not a bare string (fixture g13_smooth_tile_ndvi / g22_smooth_radar_tile_smi).
func (i Image) Reproject(crs string, scale float64) Image {
	return Image{Invoke("Image.reproject", map[string]Node{
		"image": i.N, "crs": Projection(crs), "scale": Const(scale),
	})}
}

func (i Image) FocalMean(radius float64, kernelType, units string) Image {
	return Image{Invoke("Image.focal_mean", map[string]Node{
		"image": i.N, "radius": Const(radius),
		"kernelType": Const(kernelType), "units": Const(units),
	})}
}

func (i Image) FocalMedian(radius float64, kernelType, units string) Image {
	return Image{Invoke("Image.focal_median", map[string]Node{
		"image": i.N, "radius": Const(radius),
		"kernelType": Const(kernelType), "units": Const(units),
	})}
}

// Visualize folds min/max/palette into the graph. This is how getMapId works:
// the maps request body carries NO visualizationOptions, only the visualised
// expression (verified against a live call — see §5.6 and
// testdata/maps_request_response.json).
func (i Image) Visualize(min, max float64, palette []string) Image {
	return Image{Invoke("Image.visualize", map[string]Node{
		"image":   i.N,
		"min":     Const(min),
		"max":     Const(max),
		"palette": Const(palette),
	})}
}

// PixelArea is ee.Image.pixelArea().
func PixelArea() Image {
	return Image{Invoke("Image.pixelArea", map[string]Node{})}
}

func (i Image) ReduceRegion(reducer Reducer, geom Node, scale float64, maxPixels float64) Node {
	args := map[string]Node{
		"image":    i.N,
		"reducer":  reducer.N,
		"geometry": geom,
		"scale":    Const(scale),
	}
	if maxPixels > 0 {
		args["maxPixels"] = Const(maxPixels)
	}
	return Invoke("Image.reduceRegion", args)
}

// CopyProperties mirrors image.copyProperties(source, properties).
//
// The algorithm is "Image.copyProperties", NOT "Element.copyProperties":
// although copyProperties is defined on ee.Element, the serialiser resolves it
// against the receiver's concrete type (fixture g18_s1_speckle_collection).
func (i Image) CopyProperties(source Image, properties []string) Image {
	return Image{Invoke("Image.copyProperties", map[string]Node{
		"destination": i.N,
		"source":      source.N,
		"properties":  Const(properties),
	})}
}

// ── ImageCollection ─────────────────────────────────────────────────────────

type Collection struct{ N Node }

func Coll(n Node) Collection { return Collection{N: n} }

func ImageCollectionLoad(id string) Collection {
	return Collection{Invoke("ImageCollection.load", map[string]Node{"id": Const(id)})}
}

// Filter is Collection.filter. filterBounds, filterDate and filterMetadata all
// serialise through this one function with a different Filter.* argument — the
// Python client's convenience methods are not separate EE algorithms.
func (c Collection) Filter(f Filter) Collection {
	return Collection{Invoke("Collection.filter", map[string]Node{
		"collection": c.N, "filter": f.N,
	})}
}

func (c Collection) FilterBounds(geom Geometry) Collection {
	return c.Filter(FilterIntersects(geom))
}

func (c Collection) FilterDate(start, end string) Collection {
	return c.Filter(FilterDateRange(start, end))
}

// Select on a collection is NOT an "ImageCollection.select" algorithm. The
// Python client implements it as .map(img => img.select(bands)), which is what
// the wire format shows (fixture g17_s1_base_collection). Emitting a
// non-existent ImageCollection.select would produce a runtime 400 from Google.
func (c Collection) Select(b *Builder, bands ...string) Collection {
	return c.Map(b, func(img Image) Image { return img.Select(bands...) })
}

func (c Collection) Size() Node {
	return Invoke("Collection.size", map[string]Node{"collection": c.N})
}

// Median reduces the collection to a median composite.
func (c Collection) Median() Image {
	return Image{Invoke("reduce.median", map[string]Node{"collection": c.N})}
}

func (c Collection) AggregateMean(property string) Node {
	return Invoke("AggregateFeatureCollection.mean", map[string]Node{
		"collection": c.N, "property": Const(property),
	})
}

func (c Collection) AggregateArray(property string) Node {
	return Invoke("AggregateFeatureCollection.array", map[string]Node{
		"collection": c.N, "property": Const(property),
	})
}

// Map applies a client-side body function over the collection.
//
// PRD §5.8: the serialiser turns a Python lambda into a functionDefinitionValue
// whose argumentNames are auto-generated placeholders and whose body is a
// reference to a subgraph where that variable appears as an argumentReference.
// The argument key is `baseAlgorithm` and the name convention is
// _MAPPING_VAR_<depth>_<index>, both confirmed from fixtures G1/G9/G14/G18.
func (c Collection) Map(b *Builder, fn func(element Image) Image) Collection {
	name := b.pushVar()
	defer b.popVar()
	body := fn(Image{ArgRef(name)})
	return Collection{Invoke("Collection.map", map[string]Node{
		"collection":    c.N,
		"baseAlgorithm": FuncDef([]string{name}, body.N),
	})}
}

// MapToFeature is Map for a body that returns a Feature rather than an Image.
func (c Collection) MapToFeature(b *Builder, fn func(element Image) Node) Collection {
	name := b.pushVar()
	defer b.popVar()
	body := fn(Image{ArgRef(name)})
	return Collection{Invoke("Collection.map", map[string]Node{
		"collection":    c.N,
		"baseAlgorithm": FuncDef([]string{name}, body),
	})}
}

// MapFeatures is Map over a FeatureCollection, where the loop variable is a
// Feature (used by the grid reduction, G9/G23).
func (c Collection) MapFeatures(b *Builder, fn func(element Feature) Feature) Collection {
	name := b.pushVar()
	defer b.popVar()
	body := fn(Feature{ArgRef(name)})
	return Collection{Invoke("Collection.map", map[string]Node{
		"collection":    c.N,
		"baseAlgorithm": FuncDef([]string{name}, body.N),
	})}
}

// ── Feature ─────────────────────────────────────────────────────────────────

type Feature struct{ N Node }

// NewFeature is ee.Feature(geometry, properties). A nil geometry is ee.Feature(None, …).
func NewFeature(geom Node, props Node) Feature {
	args := map[string]Node{}
	if geom != nil {
		args["geometry"] = geom
	}
	if props != nil {
		args["metadata"] = props
	}
	return Feature{Invoke("Feature", args)}
}

func (f Feature) Geometry() Geometry {
	return Geometry{Invoke("Feature.geometry", map[string]Node{"feature": f.N})}
}

// SetDict attaches a whole dictionary of properties at once.
//
// The Python code calls cell.set(reduceRegion_result) with a SINGLE dictionary
// argument, which the client resolves to "Element.setMulti" — not
// "Element.set", which is the (name, value) form. The argument key is
// "properties". Both were read from fixture g09_grid_reduced.
func (f Feature) SetDict(d Node) Feature {
	return Feature{Invoke("Element.setMulti", map[string]Node{
		"object":     f.N,
		"properties": d,
	})}
}

// ── Geometry / Projection ───────────────────────────────────────────────────

type Geometry struct{ N Node }

func Geom(n Node) Geometry { return Geometry{N: n} }

// Polygon builds GeometryConstructors.Polygon from GeoJSON rings.
func Polygon(rings [][][]float64) Geometry {
	ringNodes := make([]Node, len(rings))
	for i, ring := range rings {
		pts := make([]Node, len(ring))
		for j, p := range ring {
			pts[j] = Const(p)
		}
		ringNodes[i] = Array(pts...)
	}
	return Geometry{Invoke("GeometryConstructors.Polygon", map[string]Node{
		"coordinates": Array(ringNodes...),
	})}
}

func Point(lng, lat float64) Geometry {
	return Geometry{Invoke("GeometryConstructors.Point", map[string]Node{
		"coordinates": Const([]float64{lng, lat}),
	})}
}

func (g Geometry) CoveringGrid(proj Node) Collection {
	return Collection{Invoke("Geometry.coveringGrid", map[string]Node{
		"geometry": g.N, "proj": proj,
	})}
}

// Projection is ee.Projection(crs).
func Projection(crs string) Node {
	return Invoke("Projection", map[string]Node{"crs": Const(crs)})
}

// AtScale is projection.atScale(metres).
func AtScale(proj Node, metres float64) Node {
	return Invoke("Projection.atScale", map[string]Node{
		"projection": proj, "meters": Const(metres),
	})
}

// ── Filter ──────────────────────────────────────────────────────────────────

type Filter struct{ N Node }

// FilterIntersects is what .filterBounds(geom) serialises to. The leftField
// ".all" is literal.
func FilterIntersects(geom Geometry) Filter {
	return Filter{Invoke("Filter.intersects", map[string]Node{
		"leftField":  Const(".all"),
		"rightValue": NewFeature(geom.N, nil).N,
	})}
}

// FilterDateRange is what .filterDate(start, end) serialises to. The window is
// end-EXCLUSIVE (§7.1).
func FilterDateRange(start, end string) Filter {
	return Filter{Invoke("Filter.dateRangeContains", map[string]Node{
		"leftValue": Invoke("DateRange", map[string]Node{
			"start": Const(start), "end": Const(end),
		}),
		"rightField": Const("system:time_start"),
	})}
}

func FilterLessThan(field string, value any) Filter {
	return Filter{Invoke("Filter.lessThan", map[string]Node{
		"leftField": Const(field), "rightValue": Const(value),
	})}
}

func FilterEquals(field string, value any) Filter {
	return Filter{Invoke("Filter.equals", map[string]Node{
		"leftField": Const(field), "rightValue": Const(value),
	})}
}

func FilterListContains(field string, value any) Filter {
	return Filter{Invoke("Filter.listContains", map[string]Node{
		"leftField": Const(field), "rightValue": Const(value),
	})}
}

// ── Reducer ─────────────────────────────────────────────────────────────────

type Reducer struct{ N Node }

func ReducerMean() Reducer   { return Reducer{Invoke("Reducer.mean", map[string]Node{})} }
func ReducerStdDev() Reducer { return Reducer{Invoke("Reducer.stdDev", map[string]Node{})} }
func ReducerFirst() Reducer  { return Reducer{Invoke("Reducer.first", map[string]Node{})} }
func ReducerSum() Reducer    { return Reducer{Invoke("Reducer.sum", map[string]Node{})} }

// Group is reducer.group(groupField, groupName), used by the NDVI area
// histogram (G12).
func (r Reducer) Group(groupField int, groupName string) Reducer {
	return Reducer{Invoke("Reducer.group", map[string]Node{
		"reducer":    r.N,
		"groupField": Const(groupField),
		"groupName":  Const(groupName),
	})}
}

// ── Date / List ─────────────────────────────────────────────────────────────

func DateFormat(millis Node, pattern string) Node {
	return Invoke("Date.format", map[string]Node{
		"date":   Invoke("Date", map[string]Node{"value": millis}),
		"format": Const(pattern),
	})
}

// GetProperty is element.get("name").
func GetProperty(el Node, name string) Node {
	return Invoke("Element.get", map[string]Node{
		"object": el, "property": Const(name),
	})
}

func ListDistinct(list Node) Node {
	return Invoke("List.distinct", map[string]Node{"list": list})
}

func ListSort(list Node) Node {
	return Invoke("List.sort", map[string]Node{"list": list})
}

// ── Builder ─────────────────────────────────────────────────────────────────

// Builder tracks function-definition nesting so that mapping variables get the
// same names the Python serialiser produces.
//
// Both .map() calls in fixture G1 use _MAPPING_VAR_0_0: the counter is per
// nesting depth, not global. A map nested inside another map's body would be
// _MAPPING_VAR_1_0.
type Builder struct{ depth int }

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) pushVar() string {
	name := fmt.Sprintf("_MAPPING_VAR_%d_0", b.depth)
	b.depth++
	return name
}

func (b *Builder) popVar() { b.depth-- }
