# frozen_string_literal: true

require 'colorize'
require_relative 'utils'

RSpec.describe :link do
  dotty = Dotty.new

  it 'can link to a file' do
    src = Pathname.new('foo')
    dst = Pathname.new('bar')
    dotty.in_config do
      src.open('w')
      expect(src).to exist
    end

    dotty_run_script '((:link "foo" "~/bar"))', dotty do
      dotty.in_home do
        expect(dst).to exist
        expect(dst.symlink?).to be(true), "#{dst} is not a symlink"
        expect(dst.readlink).to eq(Pathname.new('') / dotty.config_dir / src)
      end
    end
  end

  it 'can link to a dir' do
    src = Pathname.new('foo')
    dst = Pathname.new('bar')
    dotty.in_config do
      src.mkdir
      expect(src).to exist
      expect(src).to be_directory
    end

    dotty_run_script '((:link "foo" "~/bar"))', dotty do
      dotty.in_home do
        expect(dst).to exist
        expect(dst).to be_directory
        expect(dst.symlink?).to be(true), "#{dst} is not a symlink"
        expect(dst.readlink).to eq(Pathname.new('') / dotty.config_dir / src)
      end
    end
  end

  it 'can glob for sources' do
    dst = Pathname.new('foo')
    srcs = %w[bar baz bag fob].map(&Pathname.method(:new))
    glob = 'ba*'
    dotty.in_config do
      srcs.each { |src| src.open('w'); expect(src).to exist }
    end

    dotty_run_script '((:link {:src "ba*" :dest "~/foo" :glob true}))', dotty do
      dotty.in_home do
        srcs.each do |src|
          full_path = dst / src
          if src.fnmatch(glob)
            expect(full_path).to exist
            expect(full_path.symlink?).to be(true), "#{src} is not a symlink"
            expect(full_path.readlink).to eq(Pathname.new('') / dotty.config_dir / src)
          else
            expect(full_path).to_not exist
          end
        end
      end
    end
  end

  it 'can link into a dir' do
    # destinations with a trailing slash point into directories
    src = Pathname.new('foo')
    dotty.in_config do
      src.open('w')
      expect(src).to exist
    end

    dst_dir = Pathname.new('bar')
    dotty_run_script '((:link "foo" "~/bar/"))', dotty do
      dotty.in_home do
        dst = dst_dir / src

        expect(dst_dir).to exist
        expect(dst_dir).to be_directory
        expect(dst).to exist
        expect(dst.symlink?).to be(true), "#{dst} is not a symlink"
        expect(dst.readlink).to eq(Pathname.new('') / dotty.config_dir / src)
      end
    end
  end

  it 'can link multiple sources into a single destination' do
    dst = Pathname.new('bag')
    srcs = %w[foo bar baz].map(&Pathname.method(:new))
    dotty.in_config do
      srcs.each { |src| src.open('w'); expect(src).to exist }
    end

    dotty_run_script '((:link ("foo" "bar" "baz") "~/bag"))', dotty do
      dotty.in_home do
        expect(dst).to exist
        expect(dst).to be_directory
        srcs.each do |src|
          path = dst / src
          expect(path).to exist
          expect(path.symlink?).to be(true), "#{path} is not a symlink"
          expect(path.readlink).to eq(Pathname.new('') / dotty.config_dir / src)
        end
      end
    end
  end

  it 'can link a src into a multiple destinations' do
    src = Pathname.new('foo')
    dsts = %w[bar baz bag].map(&Pathname.method(:new))
    dotty.in_config { src.open('w'); expect(src).to exist; }

    dotty_run_script '((:link "foo" ("~/bar" "~/baz" "~/bag")))', dotty do
      dotty.in_home do
        dsts.each do |dst|
          expect(dst).to exist
          expect(dst.symlink?).to be(true), "#{dst} is not a symlink"
          expect(dst.readlink).to eq(Pathname.new('') / dotty.config_dir / src)
        end
      end
    end
  end

  it 'can link a multiple sources into a multiple destinations' do
    srcs = %w[foo bar baz].map(&Pathname.method(:new))
    dsts = %w[bag bam bat].map(&Pathname.method(:new))
    dotty.in_config do
      srcs.each { |src| src.open('w'); expect(src).to exist }
    end

    dotty_run_script '((:link ("foo", "bar", "baz") ("~/bag" "~/bam" "~/bat")))', dotty do
      dotty.in_home do
        dsts.each do |dst|
          expect(dst).to exist
          expect(dst).to be_directory

          srcs.each do |src|
            path = dst / src
            expect(path).to exist
            expect(path.symlink?).to be(true), "#{path} is not a symlink"
            expect(path.readlink).to eq(Pathname.new('') / dotty.config_dir / src)
          end
        end
      end
    end
  end

  it "automatically links into destination if it's a directory" do
    src = Pathname.new('foo')
    dst = Pathname.new('bar')
    dotty.in_config { src.open('w'); expect(src).to exist }
    dotty.in_home do
      dst.mkdir
      expect(dst).to exist
      expect(dst).to be_directory
    end

    dotty_run_script '((:link "foo" "~/bar"))', dotty do
      dotty.in_home do
        # make sure dotty didn't overwrite the destination
        expect(dst).to exist
        expect(dst).to be_directory

        path = dst / src
        expect(path).to exist
        expect(path.symlink?).to be(true), "#{path} is not a symlink"
        expect(path.readlink).to eq(Pathname.new('') / dotty.config_dir / src)
      end
    end
  end

  it 'can relink existing symlinks' do
    src1 = Pathname.new('foo')
    src2 = Pathname.new('bar')
    dst = Pathname.new('baz')
    dotty.in_config do
      # create both src files
      [src1, src2].each { |src| src.open('w'); expect(src1).to exist }
    end
    dotty.in_home do
      full_src = Pathname.new('') / dotty.config_dir / src1
      dst.make_symlink(full_src)
      expect(dst).to exist
      expect(dst.symlink?).to be(true), "#{dst} is not a symlink"
      expect(dst.readlink).to eq(full_src)
    end

    dotty_run_script '((:link {:src "bar" :dest "~/baz" :relink true}))', dotty do |_, _, _, _serr|
      dotty.in_home do
        expect(dst).to exist
        expect(dst.symlink?).to be(true), "#{dst} is not a symlink"
        expect(dst.readlink).to eq(Pathname.new('') / dotty.config_dir / src2)
      end
    end
  end

  context 'making links with missing src' do
    it 'can make broken symlinks' do
      src = Pathname.new('foo')
      dst = Pathname.new('bar')
      dotty.in_config do
        expect(src).to_not exist
      end

      dotty_run_script '((:link {:src "foo" :dest "~/bar" :ignore-missing true}))', dotty do
        dotty.in_home do
          expect { dst.lstat }.to_not raise_error
          expect(dst.symlink?).to be(true), "#{dst} is not a symlink"
          expect(dst.readlink).to eq(Pathname.new('') / dotty.config_dir / src)
        end
      end
    end

    it 'cannot make broken hardlinks' do
      src = Pathname.new('foo')
      expect(src).not_to exist

      dotty.script '((:link "foo" "~/bar"))'
      dotty.run_wait do |_, _, serr, proc|
        err = serr.read
        expect(proc.to_i).not_to eq(0), err
        expect(err.uncolorize).to match(/ERR Link src not found/)
      end
    ensure
      dotty.cleanup
    end
  end

  context 'making hard links' do
    it 'can link to a file' do
      src = Pathname.new('foo')
      dst = Pathname.new('bar')
      dotty.in_config do
        src.open('w')
        expect(src).to exist
      end

      dotty_run_script '((:link {:src "foo" :dest "~/bar" :symbolic false}))', dotty do
        dotty.in_home do
          expect(dst).to exist
          expect(dst.stat.nlink).to be >= 2
        end
      end
    end

    it 'cannot link to a directory' do
      src = Pathname.new('foo')
      dotty.in_config { src.mkdir }

      dotty.script '((:link {:src "foo" :dest "~/bar" :symbolic false}))'
      dotty.run_wait do |_, _, serr, proc|
        err = serr.read
        expect(proc.to_i).not_to eq(0), err
        expect(err.uncolorize).to match(/ERR Failed to link files/)
      end
    ensure
      dotty.cleanup
    end
  end

  it "doesn't overwrite existing files" do
    src = Pathname.new('foo')
    dst = Pathname.new('bar')
    dotty.in_config do
      src.open('w')
      expect(src).to exist
    end
    dotty.in_home do
      dst.open('w')
      expect(dst).to exist
    end

    dotty_run_script '((:link "foo" "~/bar"))', dotty do |_, _, _, serr, _proc|
      err = serr.read
      expect(err.uncolorize).to match(/DBG Skipping linking src to dest because dest exists/), err
    end
  end

  context 'force is true' do
    it 'can overwrite destination files' do
      src = Pathname.new('foo')
      dst = Pathname.new('bar')
      dotty.in_config do
        src.open('w')
        expect(src).to exist
      end
      dotty.in_home do
        dst.open('w')
        expect(dst).to exist
      end

      dotty_run_script '((:link {:src "foo" :dest "~/bar" :force true}))', dotty do
        dotty.in_home do
          expect { dst.lstat }.to_not raise_error
          expect(dst.symlink?).to be(true), "#{dst} is not a symlink"
          expect(dst.readlink).to eq(Pathname.new('') / dotty.config_dir / src)
        end
      end
    end

    it "doesn't overwrite directories" do
      src = Pathname.new('foo')
      dst = Pathname.new('bar')
      dotty.in_config do
        src.open('w')
        expect(src).to exist
      end
      dotty.in_home do
        dst.mkdir
        expect(dst).to exist
        expect(dst).to be_directory
      end

      dotty_run_script '((:link {:src "foo" :dest "~/bar" :force true}))' do |_, _, _, serr, _proc|
        err = serr.read
        expect(err.uncolorize).to match(/WRN Skipping force link because dest is a directory/)
      end
    end
  end
end
