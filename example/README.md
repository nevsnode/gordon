Goophry Example Configuration
=============================

Have a look at the files in this directory, to see how things are connected to each other.

___

**addtask.php** is an example on how to use the class in `goophry.php` to create tasks.

**goophry.config.json** is the configuration file for the *goophry* binary in which the task `sometask` is configured.
Goophry will execute the script `somescript.php` with the parameters as they are provided in `addtask.php`.

**somescript.php** is the script which is executed from Goophry for the task-type `sometask`.
It requires one parameter, as it will be provided in `addtask.php`.
