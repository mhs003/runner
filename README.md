Runner
======

`Runner` is a command runner script, that helps to run some specific commands in current working directory 

## install

In **Linux**,

Clone the repository
```bash
$ git clone https://github.com/mhs003/runner.git
```

then run,
```bash
$ cd runner
$ cp ./run /usr/local/bin
$ sudo chmod +x /usr/local/bin/run
```

## how to use

Create a `.runner` file in your working directory. Place your commands in the file.

Example:

```js
/* `main` is the default command */
main: python3 server.py
test: python3 test.py
js: node index.js
test: npm run test
/* command_name: regular command */
```

then run in command-line:

```yaml
$ run
# will execute `python3 server.py`

$ run test
# will execute `python3 test.py` and will skip `npm run test`

$ run js
# will execute `node index.js`

$ run main
# will execute `python3 server.py`
```
