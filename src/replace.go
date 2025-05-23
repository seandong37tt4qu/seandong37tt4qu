package f2

import (
	"errors"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type numbersToSkip struct {
	min int
	max int
}

type numberVar struct {
	submatches [][]string
	values     []struct {
		regex       *regexp.Regexp
		startNumber int
		index       string
		format      string
		step        int
		skip        []numbersToSkip
	}
}

type transformVar struct {
	submatches [][]string
	values     []struct {
		regex *regexp.Regexp
		token string
	}
}

type exiftoolVar struct {
	submatches [][]string
	values     []struct {
		regex *regexp.Regexp
		attr  string
	}
}

type exifVar struct {
	submatches [][]string
	values     []struct {
		regex   *regexp.Regexp
		attr    string
		timeStr string
	}
}

type id3Var struct {
	submatches [][]string
	values     []struct {
		regex *regexp.Regexp
		tag   string
	}
}

type dateVar struct {
	submatches [][]string
	values     []struct {
		regex *regexp.Regexp
		attr  string
		token string
	}
}

type hashVar struct {
	submatches [][]string
	values     []struct {
		regex  *regexp.Regexp
		hashFn hashAlgorithm
	}
}

type randomVar struct {
	submatches [][]string
	values     []struct {
		regex      *regexp.Regexp
		length     int
		characters string
	}
}

type replaceVars struct {
	exif      exifVar
	exiftool  exiftoolVar
	number    numberVar
	id3       id3Var
	hash      hashVar
	date      dateVar
	random    randomVar
	transform transformVar
}

var (
	errInvalidSubmatches = errors.New("Invalid number of submatches")
)

func getDateVar(str string) (dateVar, error) {
	var d dateVar
	if dateRegex.MatchString(str) {
		d.submatches = dateRegex.FindAllStringSubmatch(str, -1)
		expectedLength := 3

		for _, submatch := range d.submatches {
			if len(submatch) < expectedLength {
				return d, errInvalidSubmatches
			}

			var x struct {
				regex *regexp.Regexp
				attr  string
				token string
			}
			regex, err := regexp.Compile(submatch[0])
			if err != nil {
				return d, err
			}

			x.regex = regex
			x.attr = submatch[1]
			x.token = submatch[2]

			d.values = append(d.values, x)
		}
	}

	return d, nil
}

// getHashVar retrieves all the hash variables in the replacement
// string if any
func getHashVar(str string) (hashVar, error) {
	var h hashVar
	if hashRegex.MatchString(str) {
		h.submatches = hashRegex.FindAllStringSubmatch(str, -1)
		expectedLength := 2

		for _, submatch := range h.submatches {
			if len(submatch) < expectedLength {
				return h, errInvalidSubmatches
			}

			var x struct {
				regex  *regexp.Regexp
				hashFn hashAlgorithm
			}
			regex, err := regexp.Compile(submatch[0])
			if err != nil {
				return h, err
			}

			x.regex = regex
			x.hashFn = hashAlgorithm(submatch[1])
			h.values = append(h.values, x)
		}
	}

	return h, nil
}

// getTransformVar retrieves all the string transformation variables
// in the replacement string if any
func getTransformVar(str string) (transformVar, error) {
	var t transformVar
	if transformRegex.MatchString(str) {
		t.submatches = transformRegex.FindAllStringSubmatch(str, -1)
		expectedLength := 2

		for _, submatch := range t.submatches {
			if len(submatch) < expectedLength {
				return t, errInvalidSubmatches
			}

			var x struct {
				regex *regexp.Regexp
				token string
			}
			regex, err := regexp.Compile(submatch[0])
			if err != nil {
				return t, err
			}

			x.regex = regex
			x.token = submatch[1]
			t.values = append(t.values, x)
		}
	}

	return t, nil
}

// getExifVar retrieves all the exif variables in the replacement
// string if any
func getExifVar(str string) (exifVar, error) {
	var ex exifVar

	if exifRegex.MatchString(str) {
		ex.submatches = exifRegex.FindAllStringSubmatch(str, -1)
		expectedLength := 3

		for _, submatch := range ex.submatches {
			if len(submatch) < expectedLength {
				return ex, errInvalidSubmatches
			}

			var x struct {
				regex   *regexp.Regexp
				attr    string
				timeStr string
			}
			regex, err := regexp.Compile(submatch[0])
			if err != nil {
				return ex, err
			}

			x.regex = regex

			if strings.Contains(submatch[0], "exif.dt") ||
				strings.Contains(submatch[0], "x.dt") {
				submatch = append(submatch[:1], submatch[1+1:]...)
			}

			x.attr = submatch[1]
			if x.attr == "dt" {
				x.timeStr = submatch[2]
			}

			ex.values = append(ex.values, x)
		}
	}

	return ex, nil
}

// getNumberVar retrieves all the index variables in the replacement string
// if any
func getNumberVar(str string) (numberVar, error) {
	var nv numberVar

	if indexRegex.MatchString(str) {
		nv.submatches = indexRegex.FindAllStringSubmatch(str, -1)
		expectedLength := 7

		for _, submatch := range nv.submatches {
			if len(submatch) < expectedLength {
				return nv, errInvalidSubmatches
			}

			var n struct {
				regex       *regexp.Regexp
				startNumber int
				index       string
				format      string
				step        int
				skip        []numbersToSkip
			}

			regex, err := regexp.Compile(submatch[0])
			if err != nil {
				return nv, err
			}

			n.regex = regex

			if submatch[1] != "" {
				n.startNumber, err = strconv.Atoi(submatch[1])
				if err != nil {
					return nv, err
				}
			} else {
				n.startNumber = 1
			}

			n.index = submatch[2]
			n.format = submatch[4]
			n.step = 1
			if submatch[5] != "" {
				n.step, err = strconv.Atoi(submatch[5])
				if err != nil {
					return nv, err
				}
			}

			skipNumbers := submatch[6]
			if skipNumbers != "" {
				slice := strings.Split(skipNumbers, ",")
				for _, v := range slice {
					if strings.Contains(v, "-") {
						sl := strings.Split(v, "-")
						n1, err := strconv.Atoi(sl[0])
						if err != nil {
							return nv, err
						}

						n2, err := strconv.Atoi(sl[1])
						if err != nil {
							return nv, err
						}

						n.skip = append(n.skip, numbersToSkip{
							max: int(math.Max(float64(n1), float64(n2))),
							min: int(math.Min(float64(n1), float64(n2))),
						})
						continue
					}

					num, err := strconv.Atoi(v)
					if err != nil {
						return nv, err
					}

					n.skip = append(n.skip, numbersToSkip{
						max: num,
						min: num,
					})
				}
			}

			nv.values = append(nv.values, n)
		}
	}

	return nv, nil
}

// getExifToolVar retrieves all the exiftool variables in the
// replacement string if any
func getExifToolVar(str string) (exiftoolVar, error) {
	var et exiftoolVar
	if exiftoolRegex.MatchString(str) {
		et.submatches = exiftoolRegex.FindAllStringSubmatch(str, -1)
		expectedLength := 2

		for _, submatch := range et.submatches {
			if len(submatch) < expectedLength {
				return et, errInvalidSubmatches
			}

			var x struct {
				regex *regexp.Regexp
				attr  string
			}
			regex, err := regexp.Compile(submatch[0])
			if err != nil {
				return et, err
			}

			x.regex = regex
			x.attr = submatch[1]

			et.values = append(et.values, x)
		}
	}

	return et, nil
}

// getID3Var retrieves all the id3 variables in the
// replacement string if any
func getID3Var(str string) (id3Var, error) {
	var iv id3Var
	if id3Regex.MatchString(str) {
		iv.submatches = id3Regex.FindAllStringSubmatch(str, -1)
		expectedLength := 2

		for _, submatch := range iv.submatches {
			if len(submatch) < expectedLength {
				return iv, errInvalidSubmatches
			}

			var x struct {
				regex *regexp.Regexp
				tag   string
			}
			regex, err := regexp.Compile(submatch[0])
			if err != nil {
				return iv, err
			}

			x.regex = regex
			x.tag = submatch[1]

			iv.values = append(iv.values, x)
		}
	}

	return iv, nil
}

// getRandomVar retrieves all the random variables in the
// replacement string if any
func getRandomVar(str string) (randomVar, error) {
	var rv randomVar

	if randomRegex.MatchString(str) {
		rv.submatches = randomRegex.FindAllStringSubmatch(str, -1)
		expectedLength := 4

		for _, submatch := range rv.submatches {
			if len(submatch) < expectedLength {
				return rv, errInvalidSubmatches
			}

			var val struct {
				regex      *regexp.Regexp
				length     int
				characters string
			}
			val.length = 10
			regex, err := regexp.Compile(submatch[0])
			if err != nil {
				return rv, err
			}
			val.regex = regex

			strLen := submatch[1]
			if strLen != "" {
				val.length, err = strconv.Atoi(strLen)
				if err != nil {
					return rv, err
				}
			}

			val.characters = submatch[2]

			if submatch[3] != "" {
				val.characters = submatch[3]
			}

			rv.values = append(rv.values, val)
		}
	}

	return rv, nil
}

// getAllVariables retrieves all the variables present in the replacement
// string
func getAllVariables(str string) (replaceVars, error) {
	var v replaceVars
	var err error
	v.exif, err = getExifVar(str)
	if err != nil {
		return v, err
	}

	v.number, err = getNumberVar(str)
	if err != nil {
		return v, err
	}

	v.id3, err = getID3Var(str)
	if err != nil {
		return v, err
	}

	v.hash, err = getHashVar(str)
	if err != nil {
		return v, err
	}

	v.date, err = getDateVar(str)
	if err != nil {
		return v, err
	}

	v.random, err = getRandomVar(str)
	if err != nil {
		return v, err
	}

	v.exiftool, err = getExifToolVar(str)
	if err != nil {
		return v, err
	}

	v.transform, err = getTransformVar(str)
	if err != nil {
		return v, err
	}

	return v, nil
}

// regexReplace replaces matches in the filename with the replacement string.
// It respects the specified replacement limit. A negative limit starts the
// replacement from the end of the fileName
func regexReplace(
	r *regexp.Regexp,
	fileName, replacement string,
	replaceLimit int,
) string {
	var output string

	switch limit := replaceLimit; {
	case limit > 0:
		counter := 0
		output = r.ReplaceAllStringFunc(
			fileName,
			func(val string) string {
				if counter == replaceLimit {
					return val
				}

				counter++
				return r.ReplaceAllString(val, replacement)
			},
		)
	case limit < 0:
		matches := r.FindAllString(fileName, -1)

		l := len(matches) + limit
		counter := 0
		output = r.ReplaceAllStringFunc(
			fileName,
			func(val string) string {
				if counter >= l {
					return r.ReplaceAllString(val, replacement)
				}

				counter++
				return val
			},
		)
	default:
		output = r.ReplaceAllString(fileName, replacement)
	}

	return output
}

// replaceString replaces all matches in the filename
// with the replacement string
func (op *Operation) replaceString(fileName string) (str string) {
	return regexReplace(
		op.searchRegex,
		fileName,
		op.replacement,
		op.replaceLimit,
	)
}

// replace replaces the matched text in each path with the
// replacement string
func (op *Operation) replace() (err error) {
	vars, err := getAllVariables(op.replacement)
	if err != nil {
		return err
	}

	for i, v := range op.matches {
		fileName := v.Source
		fileExt := filepath.Ext(fileName)
		if op.ignoreExt {
			fileName = filenameWithoutExtension(fileName)
		}

		str := op.replaceString(fileName)

		// handle variables
		str, err = op.handleVariables(str, v, &vars)
		if err != nil {
			return err
		}

		// If numbering scheme is present
		if indexRegex.MatchString(str) {
			str = op.replaceIndex(str, i, vars.number)
		}

		if op.ignoreExt {
			str += fileExt
		}

		v.Target = strings.TrimSpace(filepath.Join(str))
		op.matches[i] = v
	}

	return nil
}
