package com.tunnelmanager.app;

import java.util.List;

public final class AssistantTaskSelectionTest {
    public static void main(String[] args) {
        AssistantTaskSelection selection = new AssistantTaskSelection();
        selection.sync("first", List.of("task-a", "task-b"));
        switch (args[0]) {
            case "explicit":
                check(selection.snapshot("first", false).isEmpty(), "tasks selected without consent");
                selection.select("missing", true);
                check(selection.size() == 0, "unknown task selected");
                selection.select("task-a", true);
                selection.select("task-a", true);
                check(selection.snapshot("first", false).equals(List.of("task-a")), "duplicate selection");
                selection.select("task-a", false);
                check(selection.size() == 0, "uncheck ignored");
                break;
            case "refresh":
                selection.select("task-a", true);
                selection.select("task-b", true);
                selection.sync("first", List.of("task-a", "task-b", "task-c"));
                check(selection.size() == 2 && !selection.contains("task-c"), "refresh changed consent");
                selection.sync("first", List.of("task-b"));
                check(selection.snapshot("first", false).equals(List.of("task-b")), "finished task still selected");
                break;
            case "isolation":
                selection.select("task-a", true);
                check(selection.snapshot("second", false).isEmpty(), "selection leaked across conversations");
                check(selection.snapshot("first", true).isEmpty(), "execution allowed while busy");
                selection.sync("second", List.of("task-a"));
                check(selection.size() == 0, "conversation switch retained selection");
                selection.sync("", List.of("task-a"));
                selection.select("task-a", true);
                check(selection.snapshot("", false).isEmpty(), "empty conversation can execute");
                break;
            case "snapshot":
                selection.select("task-b", true);
                selection.select("task-a", true);
                List<String> approved = selection.snapshot("first", false);
                selection.clear();
                selection.sync("second", List.of());
                check(approved.equals(List.of("task-b", "task-a")), "confirmation changed with live selection");
                approved.clear();
                check(selection.size() == 0, "snapshot mutated selection");
                break;
            case "labels":
                check("间隔（秒）".equals(AssistantTaskSelection.argumentLabel("interval_sec")), "untranslated interval");
                check("探测地址".equals(AssistantTaskSelection.argumentLabel("url")), "untranslated URL");
                check("future_field".equals(AssistantTaskSelection.argumentLabel("future_field")), "unknown parameter hidden");
                break;
            default: throw new AssertionError("unknown scenario");
        }
    }

    private static void check(boolean condition, String message) {
        if (!condition) throw new AssertionError(message);
    }
}
