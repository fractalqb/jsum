/*
A tool to analyse the structure of JSON from a set of example JSON values.
Copyright (C) 2025  Marcus Perlick

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package jsum

import "testing"

func TestUnknow_toBool(t *testing.T) {
	var cfg Config
	var u Deducer = NewUnknown(&cfg)
	u = u.Example(true, JsonTypeOf(true))
	if b, ok := u.(*Boolean); !ok {
		t.Fatalf("deduced not bool but %T", u)
	} else if b.TrueNo != 1 {
		t.Errorf("true count %d not 1", b.TrueNo)
	}
	u = u.Example(true, JsonTypeOf(true))
	if b, ok := u.(*Boolean); !ok {
		t.Fatalf("deduced not bool but %T", u)
	} else if b.TrueNo != 2 {
		t.Errorf("true count %d not 2", b.TrueNo)
	}
}
