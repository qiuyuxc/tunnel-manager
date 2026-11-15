package com.tunnelmanager.app;

final class SheetSizingTest {
    public static void main(String[] args) {
        switch (args[0]) {
            case "fixed-scenes": {
                SheetSizing.Geometry full = SheetSizing.geometry(1000, 1000);
                equal(full.height(SheetSizing.Tier.COMPACT), 300);
                equal(full.height(SheetSizing.Tier.MEDIUM), 600);
                equal(full.height(SheetSizing.Tier.EXPANDED), 800);
                SheetSizing.Geometry inset = SheetSizing.geometry(1000, 900);
                equal(inset.height(SheetSizing.Tier.COMPACT), 300);
                equal(inset.height(SheetSizing.Tier.MEDIUM), 600);
                equal(inset.height(SheetSizing.Tier.EXPANDED), 800);
                break;
            }
            case "keyboard-bounds": {
                SheetSizing.Tier form = SheetSizing.Tier.MEDIUM;
                equal(SheetSizing.geometry(1000, 400).height(form), 400);
                equal(SheetSizing.geometry(1000, 900).height(form), 600);
                equal(SheetSizing.geometry(1000, 400).height(SheetSizing.Tier.COMPACT), 300);
                equal(SheetSizing.geometry(1000, 200).height(SheetSizing.Tier.COMPACT), 200);
                break;
            }
            case "rotation": {
                SheetSizing.Tier scene = SheetSizing.Tier.EXPANDED;
                equal(SheetSizing.geometry(1000, 900).height(scene), 800);
                equal(SheetSizing.geometry(600, 560).height(scene), 480);
                break;
            }
            default: throw new AssertionError(args[0]);
        }
    }
    private static void equal(int value, int expected) { if (value != expected) throw new AssertionError(value + " != " + expected); }
}
