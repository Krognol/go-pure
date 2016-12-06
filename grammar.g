// Data types
UNIT ::= ( 'ms'
         | 'MB'
         | 's'
         | 'cm'
         // etc...
         ) ;
ID ::= [a-öA-Ö_-]+ ;
STRING ::= '"' .* '"' ;
INT ::= [0-9]+ ;
DOUBLE ::= INT '.' INT ;
BOOL ::= ('true' | 'false') ;
PATH ::= ('.')? (STRING | '/')+ ;
QUANTITY ::= INT UNIT ;
ENV ::= '$' ID ;
UNDEFINED ::= '[allow-undefined]' ;

VALUE ::= (STRING | INT | DOUBLE | BOOL | PATH | QUANTITY | ARRAY) ;
ARRAY ::= '[' VALUE* ']' ;

FORMAT ::= ( 'string' ('(' VALUE ')')? 
           | 'int' ('(' VALUE ')')?
           | 'double' ('(' VALUE ')')?
           | 'bool' ('(' VALUE ')')?
           | 'quantity' ('(' VALUE ')')?
           | UNDEFINED
           ) ;

TYPESPECIFIER ::= ID ':' FORMAT ; 

// Include files

INCLUDE ::= '%' ID ;

// Expressions
pureFile ::= expression* ;

expression ::= ((ID ('.' ID)* '=' VALUE) | (ID '=>' expression+)) | group ;


group ::= ID '\n' ('\t')* expression+ :