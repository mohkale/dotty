require_relative './utils'

RSpec.describe :shell do
  it "can run a subprocess" do
    dotty_run_script '((:def (:shell "interactive" true)) (:shell "echo foo"))' do |_, _, sout|
      expect(sout.read()).to match(/foo/)
    end
  end

  it "is silent by default" do
    dotty_run_script '((:shell "echo foo"))' do |_, _, sout|
      expect(sout.read()).to eq("")
    end
  end

  it "can pass cmd options through a map" do
    dotty_run_script '((:shell {:cmd "echo foo" :interactive true}))' do |_, _, sout|
      expect(sout.read()).to match(/foo/)
    end

    dotty_run_script '((:shell {:cmd ("foo=hello" "echo $foo") :interactive true}))' do |_, _, sout|
      expect(sout.read()).to match(/hello/)
    end
  end

  it "can accept multiple args as a script" do
    dotty_run_script '((:def (:shell "interactive" true)) (:shell ("foo=hello" "echo $foo")))' do |_, _, sout|
      expect(sout.read()).to match(/hello/)
    end
  end

  it "doesn't accepts multiple arguments as multiple commands" do
    dotty_run_script '((:def (:shell "interactive" true)) (:shell "foo=hello" "echo $foo"))' do |_, _, sout|
      expect(sout.read().strip()).to eq("")
    end
  end
end
