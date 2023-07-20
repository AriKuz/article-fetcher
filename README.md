# article fetcher

In this assignment, I made a system to fetch articles from a list of essays and count the top 10 words from all the
essays combined.
A valid word will:

    Contain at least 3 characters.
    Contain only alphabetic characters.
    Be part of our bank of words (not all the words in the bank are valid according to the
    previous rules)

I tried to build the system with simplicity in mind, and made the following steps in order to successfully achieve the aim of the program:

    concurrency: 
    the list of essays contains 40K URLs. in order to both fetch their content and count their combined top 10 words,
    I divided to the program to 3 different go routines: main, fetch, and process.
    I also allow, using a channel, up to 500 routines for best performance.
    
    traffic configuration:
    In order to avoid getting blocked I pre-configured several user-agents to be sent randomly in the request's header.
    Furthermore, I added MaxIdleConns, IdleConnTimeout, TLSHandshakeTimeout, ExpectContinueTimeout.

    performence:
    I placed a global context with time out and a sleep after each http request with default values
    of 15m and 4s accordingly.
    these can be configured using cmd flags
## instructions
clone the repository and run "make build", it will sync dependencies and compile the binaries for program.
then run the created binary file.
```bash
make build 
./fireFly

if running on Windows os:
go build .
run fireFly-assignment.exe
```
## notes
when running the binary, you may pass the following arguments:
```bash
-ctx-duration duration
The context time out (minutes) (default 15m0s)
-req-duration duration
System sleep between HTTP requests (seconds) (default 3s)

e.g ./fireFly -ctx-duration 20m -req-duration 2s
```
## output
should look like this with similar numbers:
```bash
{the 455460}
{and 227264}
{that 114360}
{with 74126}
{The 70356}
{you 68679}
{has 41391}
{have 40563}
{your 39916}
{from 39752}           
```
I also used zero-log package for logging in case of failures.
I noticed that a lot of links do not exist anymore so the following error will be printed to stdout:
```bash
{"level":"error","time":"2023-04-28T17:54:52+03:00","message":"https://www.engadget.com/2019/08/24/vizio-soundbars-99-today/ failed with status 404: <nil>"}   
```
