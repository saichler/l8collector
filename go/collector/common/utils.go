package common

import (
	"bytes"
	"math/rand"

)

func ReplaceArguments(what string, job *l8poll.CJob) string {
	if job.Arguments == nil {
		return what
	}
	buff := bytes.Buffer{}
	arg := bytes.Buffer{}
	open := false
	for _, c := range what {
		if c == '$' {
			open = true
		} else if c == ' ' && open {
			open = false
			v, ok := job.Arguments[arg.String()]
			if !ok {
				return what
			}
			buff.WriteString(v)
			buff.WriteString(" ")
			arg.Reset()
		} else if open {
			arg.WriteRune(c)
		} else {
			buff.WriteRune(c)
		}
	}

	if open {
		v, ok := job.Arguments[arg.String()]
		if !ok {
			return what
		}
		buff.WriteString(v)
	}
	return buff.String()
}

func RandomSecondWithin15Minutes() int {
	return rand.Intn(900)
}

func RandomSecondWithin3Minutes() int {
	return rand.Intn(180)
}
