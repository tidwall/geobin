# Geobin

The Geobin Object represents tightly packed geometry that is compatible with [GeoJSON RFC 7946](https://tools.ietf.org/html/rfc7946).


## Object layout

```
[OBJECT] >> [BBOX][EXDATA][DATA][EXDATASIZE][TAIL]
```

### TAIL

Tail is exactly one byte and is used to identify the components of the object.

```
BIT 0: 1 = GEOM, 0 = STRING
BIT 1: 1 = 3D,   0 = 2D
BIT 2: 1 = RECT, 0 = POINT
BIT 3: 1 = ISCOMPLEX
BIT 4: 1 = HASEXDATA
```

## BBOX

The BBox is always the first elements for GEOM types and is represented as a `2D/3D RECT` or `2D/3D POINT`.

```
IF STRING:
    [BBOX] >> {EMPTY}
ELSE 
    IF 3D:
        IF RECT:
            [BBOX] >> [MINX][MINY][MINZ][MAXX][MAXY][MAXZ]
        ELSE:
            [BBOX] >> [X][Y][Z]
    ELSE:
        IF RECT:
            [BBOX] >> [MINX][MINY][MAXX][MAXY]
        ELSE:
            [BBOX] >> [X][Y]
```

```
[X][Y][Z][MINX][MINY][MINZ][MAXX][MAXY][MAXZ] >> float64
```

### EXDATA

ExData is optional extra user defined data. 

```
IF !HASEXDATA:
    [EXDATA] >> {EMPTY}
ELSE
    [EXDATASIZE] >> {4 BYTES AT LEN([OBJECT])-5 -- LITTLE ENDIAN UINT32}
    [EXDATA]     >> {[EXDATASIZE] BYTES AT LEN([BBOX])}
```

### DATA

```
[DATA] = {BYTES AT LEN(BBOX)+LEN(EXDATA) TO LEN(OBJECT)-LEN(EXDATASIZE)-1}
IF STRING:
    <DONE>  # The data is the string value
ELSE IF 
    [DATA] >> [HEAD][MEMBERSIZE][MEMBERS][GEOM]
    <DONE>
```

### HEAD

Head is exactly one byte and is used to identify the components of the geometry

```
BIT 0: 1 = HASMEMBERS    # extra GeoJSON members, suchs as "id" and "properties"
BIT 1: 1 = EXPORTED_BBOX # the bbox json member should be provided on exporting
BIT 4-7: TYPE 
         0 = Unknown/Invalid
         1 = POINT
         2 = MULTIPOINT
         3 = LINESTRING
         4 = MULTILINESTRING
         5 = POLYGON
         6 = MULTIPOLYGON
         7 = GEOMETRYCOLLECTION
         8 = FEATURE
         9 = FEATURECOLLECTION
```

### MEMBERS

```
IF !HASMEMBERS:
	[MEMBERS] << {EMPTY}
ELSE
	[MEMBERSSIZE] << {VARLEN AT LEN([HEAD])}
	[MEMBERS] << {[MEMBERSSIZE] BYTES AT LEN([HEAD])+LEN([MEMBERSSIZE])}
```

### GEOM

```
IF POINT:
	IF 3D:
		[GEOM] >> [X][Y][Z]
	ELSE:
		[GEOM] >> [X][Y]
ELSE IF MULTIPOINT, LINESTRING:
	[GEOM] >> [UINT32][POINT...]
ELSE IF MULTILINESTRING, POLYGON:
	[GEOM] >> [UINT32][LINESTRING...]
ELSE IF MULTIPOLYGON:
	[GEOM] >> [UINT32][POLYGON...]
ELSE IF GEOMETRYCOLLECTION:
	[GEOM] >> [UINT32]{[UINT32][OBJECT]...}
ELSE IF FEATURE;
	[GEOM] >> [UINT32][OBJECT]
ELSE IF FEATURECOLLECTION:
	[GEOM] >> [UINT32]{[UINT32][FEATURE]...}
```
