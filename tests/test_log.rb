require 'colorize'
require_relative 'utils'

RSpec.describe :log do
  it "can log a message" do
    dotty_run_script '((:debug "hello world friend"))' do |dotty, sin, sout, serr, proc|
      expect(serr.read().uncolorize).to match(/DBG hello world friend/)
    end
  end

  it "can log at different levels" do
    msg = "hello world friend"
    script = <<-EOF
      (
       (:debug #{msg.inspect})
       (:info #{msg.inspect})
       (:warn #{msg.inspect})
      )
    EOF

    dotty_run_script script do |dotty, sin, sout, serr, proc|
      err = serr.read().uncolorize
      expect(err).to match(/DBG #{msg}/)
      expect(err).to match(/INF #{msg}/)
      expect(err).to match(/WRN #{msg}/)
    end
  end

  it "can printf log arguments" do
    dotty_run_script '((:debug "the value is %03d" 5))' do |dotty, sin, sout, serr, proc|
      expect(serr.read().uncolorize).to match(/DBG the value is 005/)
    end
  end
end
