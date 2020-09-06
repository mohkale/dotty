require_relative './utils'

RSpec.describe :mkdir do
  it "can create a directory" do
    dotty_run_script '((:mkdir "~/foo" "~/bar"))' do |dotty|
      dotty.in_home do
        ["foo", "bar"].map(&Pathname.method(:new)).each do |file|
          expect(file).to exist
          expect(file).to be_directory
        end
      end
    end
  end

  it "can create subdirectories" do
    dotty_run_script '((:mkdir "~/foo/bar/baz"))' do |dotty, _, serr|
      dotty.in_home do
        path = Pathname.new("foo/bar/baz")
        expect(path).to exist
        expect(path).to be_directory
      end
    end
  end

  it "can create directories from recursive spec" do
    dotty_run_script '((:mkdir ("~" ("foo" ("bar" "baz")))))' do |dotty|
      dotty.in_home do
        ["foo/bar", "foo/baz"].map(&Pathname.method(:new)).each do |file|
          expect(file).to exist
          expect(file).to be_directory
        end
      end
    end
  end

  it "can create directories with permissions" do
    dotty_run_script '((:mkdir {:path "~/foo/bar/baz" :chmod 700} ))' do |dotty|
      dotty.in_home do
        file = Pathname.new("foo/bar/baz")
        expect(file).to exist
        expect(file).to be_directory
        expect(file.stat.mode & 07777).to eq(0700)
      end
    end
  end

  it "can create multiple directories with permissions" do
    dotty_run_script '((:mkdir {:path ("~/foo" "~/bar") :chmod 700} ))' do |dotty|
      dotty.in_home do
        ["foo", "foo"].map(&Pathname.method(:new)).each do |file|
          expect(file).to exist
          expect(file).to be_directory
          expect(file.stat.mode & 07777).to eq(0700)
        end
      end
    end
  end

  it "doesn't mutate previous permissions" do
    dotty_run_script '((:mkdir {:path "~/foo" :chmod 700} "~/bar" ))' do |dotty|
      dotty.in_home do
        file = Pathname.new("bar")
        expect(file.stat.mode & 07777).not_to eq(0700)
      end
    end
  end

  it "can substitute environment into paths" do
    ENV['path_var'] = 'hello'

    dotty_run_script '((:mkdir "~/${path_var}/foo/bar"))' do |dotty|
      dotty.in_home do
        path = Pathname.new("hello/foo/bar")
        expect(path).to exist
        expect(path).to be_directory
      end
    end
  ensure
    ENV.delete('path_var')
  end
end
