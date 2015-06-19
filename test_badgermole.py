import unittest
from badgermole import Badgermole

class BadgermoleTester(unittest.TestCase):
	def test_badgermole(self):
		bm = Badgermole()
		bm.add_arg('foo')
		bm.add_arg('bar')
		bm.parse_args('134 hello')
		self.assertEqual(bm.args['foo'], '134')
		self.assertEqual(bm.args['bar'], 'hello')

		bm._clear()
		bm.add_arg('foo', num_args=3)
		bm.add_arg('bar')
		bm.parse_args('134 hello end_of_foo cupcake')
		self.assertEqual(bm.args['foo'], ['134', 'hello', 'end_of_foo'])

		bm._clear()
		bm.add_arg('--verbose', '-v', num_args=2)
		bm.add_arg('foo')
		bm.parse_args('-v 1 2 foo_val')
		self.assertEqual(bm.args['verbose'], ['1', '2'])
		self.assertEqual(bm.args['foo'], 'foo_val')
		bm.add_arg('--subtract', '-s', required=False)
		self.assertRaises(Exception, bm.parse_args, 'foo_val -v 1 -s')

if __name__ == '__main__':
    unittest.main()