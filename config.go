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

type DedupBool uint

const (
	DedupBoolTrue DedupBool = 1 << iota
	DedupBoolFalse
)

type DedupNumber uint

const (
	DedpuNumberIntFloat DedupNumber = 1 << iota
	DedupNumberFrac
	DedupNumberMin
	DedupNumberMax
	DedupNumberNeg
	DedupNumberPos
)

type DedupString uint

const (
	DedupStringEmpty DedupString = 1 << iota
)

type Config struct {
	DedupBool   DedupBool
	DedupNumber DedupNumber
	DedupString DedupString
}
