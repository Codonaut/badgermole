# TODO: Document usage more

print 'hi'

class Badgermole(object):
    """
    A Command line option parsing class.

    Use Badgermole.add_arg() to specify an argument to the program.
    Use Badgermole.parse_args() to parse the command line string and 
        prepare the Badgermole.args dict for use.
    """

    def __init__(self):
        self._clear()

    def _clear(self):
        self.positional = []
        self.options = {}
        self.args = {}

    def _check_name_conflicts(self, name, short_name):
        for option in self.options.values():
            if name == option.name:
                raise ValueError('Option with name {} already exists.'.format(name))
            elif short_name and short_name == option.short_name:
                raise ValueError('Option with short name {} already exists.'.format(short_name))
        for arg in self.positional:
            if name == arg.name:
                raise ValueError('Positional argument with name {} already exists.'.format(name))

    def add_arg(self,
                name, 
                short_name='',
                num_args=0,
                required=True):
        self._check_name_conflicts(name, short_name)

        positional = not (len(name) > 2 and name[:2] == '--')
        new_arg = Arg(positional, name, short_name, num_args, required)
        if positional:
            self.positional.append(new_arg)
        else:
            self.options[name] = new_arg

    def _is_option(self, s):
        for option in self.options.values():
            if option.name == s or option.short_name == s:
                return option

    def parse_args(self, command_str):
        tokens = command_str.split()
        # If an arg requires arguments expecting_args jumps into play
        expecting_args = {} 
        # Index in self.positional indicating next expected positional arg
        pos_idx = 0
        for token in tokens:
            option = self._is_option(token)
            if option:
                if expecting_args:
                    raise Exception("Expecting an argument for <{}>, but received option <{}>.".format(
                            expecting_args['arg'].out_name, option.out_name))
                if option.num_args > 0:
                    expecting_args = {
                        'option': True,
                        'arg': option,
                        'num_args': option.num_args,
                        'args': []
                    }
                else:
                    self.args[option.out_name] = True # Document that options without arguments become true
            elif expecting_args:
                expecting_args['args'].append(token)
                expecting_args['num_args'] -= 1
                if expecting_args['num_args'] == 0:
                    if not expecting_args['option']:
                        self.args[expecting_args['arg'].out_name] = expecting_args['args']
                        pos_idx += 1
                    else:
                        self.args[expecting_args['arg'].out_name] = expecting_args['args']
                    expecting_args = {}
            elif pos_idx < len(self.positional):
                # It's a positional argument
                arg = self.positional[pos_idx]
                if arg.num_args > 0 and not expecting_args:
                    expecting_args = {
                        'option': False,
                        'arg': arg,
                        'num_args': arg.num_args - 1,
                        'args': [token]
                    }
                else:
                    self.args[arg.out_name] = token
                    pos_idx += 1

        # Make sure all required arguments have been passed in
        for arg in list(self.positional + self.options.values()):
            if arg.required and arg.out_name not in self.args:
                raise Exception('Required argument <{}> was not supplied.'.format(arg.out_name))


class Arg(object):
    """
    Simple class that holds arg attributes.  It's better than using a dictionary.
    """
    def __init__(self,
                 positional,
                 name,
                 short_name,
                 num_args,
                 required):
        self.positional = positional
        self.name = name
        self.short_name = short_name
        self.num_args = num_args
        self.required = required
        if not positional:
            self.out_name = self.name[2:]
        else:
            self.out_name = self.name