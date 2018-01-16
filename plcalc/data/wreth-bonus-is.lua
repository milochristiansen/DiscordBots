
-- IN: A table with C,O,W,S keys holding the current cost total.
-- BONUS: A table like IN, holding the bonus values for all parts that have a bonus for the key this script is associated with.

IN.O = IN.O - BONUS.O

local O = BONUS.O - BONUS.O * 0.2
if O < 10.0 then
	O = BONUS.O
end

IN.O = IN.O + O

-- The return value must be a table with C,O,W,S keys holding the final resource cost.
-- If the return value is incorrect the program can crash!
return IN
