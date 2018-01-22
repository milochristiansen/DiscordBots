/*
Copyright 2018 by Milo Christiansen

This software is provided 'as-is', without any express or implied warranty. In
no event will the authors be held liable for any damages arising from the use of
this software.

Permission is granted to anyone to use this software for any purpose, including
commercial applications, and to alter it and redistribute it freely, subject to
the following restrictions:

1. The origin of this software must not be misrepresented; you must not claim
that you wrote the original software. If you use this software in a product, an
acknowledgment in the product documentation would be appreciated but is not
required.

2. Altered source versions must be plainly marked as such, and must not be
misrepresented as being the original software.

3. This notice may not be removed or altered from any source distribution.
*/

package main

import "strings"

var HelpShort = "-\nTry `!spires`, `!pattern <design ID>`, `!tweak 0c,0o,0w,0s`, `! 0c,0o,0w,0s`, or `!reload`\n" +
	"\nType `!help full` for full help, `!help ids` for a list of valid spires and patterns."

var HelpIDs = strings.Replace(`
-
Valid spire IDs:
|||
%v
|||
The following parts and patterns are available to |%v|:
|||
%v
|||
`, "|", "`", -1)

var HelpLong = strings.Replace(`
-
**Set Spire:** |!spires <list of comma separated spire IDs>|
Set the spires used to calculate COWS. The special ID |Tweak| is the "spire" used by the Tweak Production (|!tweak|) command, and |Home| is all the home spires.

In addition to specifying the entire list of spires you want, you can activate or deactivate spires with the following syntax:
|||
!spires + <list of comma separated spire IDs>
!spires - <list of comma separated spire IDs>
|||
If no spires are specified this command will print the current list.

Default spire list:
|||
Tweak, Home
|||

**Calculate Design:** |!pattern <design ID>|
Calculate the COWS for a named pattern. This pattern needs to be defined in the data files.

If you wish, you can specify a count for a pattern, or specify multiple patterns to calculate together. For example, calculate the cost of one pattern "Test-A" and two pattern "Test-B":
|||
!pattern Test-A, Test-B:2
|||
To support tinkering, you can construct temporary patterns on the fly or modify existing patterns by adding parts to an existing pattern.
|||
!pattern Test-A:2;+Part1;-Part2:3
|||

**Tweak Production:** |!tweak 0c,0o,0w,0s|
Set a modifier for spire production. For this to take effect the |@| spire must be in the spire list.

Note that the COWS numbers may be partly specified, specified in any order, or even left blank. Any missing value is set to 0.
	   
**Calculate from raw COWS:** |! 0c,0o,0w,0s|
Calculate production line COWS from a raw COWS value.

Note that the COWS numbers may be partly specified, specified in any order, or even left blank. Any missing value is set to 0.

**General Calculator:** |!calc <expression>|
Run the given Lua *expression*, and print the result. Basic math should work fine, but most common modules are not loaded and statements are not allowed, so it is pretty limited.
`, "|", "`", -1)
